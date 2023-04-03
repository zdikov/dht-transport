package api

import (
	"context"
	"dht/store"
	"encoding/json"
	"fmt"
	"github.com/anacrolix/dht/v2"
	"github.com/anacrolix/dht/v2/bep44"
	peer_store "github.com/anacrolix/dht/v2/peer-store"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/types/infohash"
	"github.com/gorilla/mux"
	"net/http"
	"net/netip"
	"net/url"
	"os"
	"strings"
	"time"
)

func NewHTTPHandler(s *dht.Server, vs *store.Memory, ps *peer_store.InMemory) *HTTPHandler {
	return &HTTPHandler{
		s:  s,
		vs: vs,
		ps: ps,
	}
}

type HTTPHandler struct {
	s  *dht.Server
	vs *store.Memory
	ps *peer_store.InMemory
}

type Item struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

func (h *HTTPHandler) Put(rw http.ResponseWriter, r *http.Request) {
	var item Item
	err := json.NewDecoder(r.Body).Decode(&item)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}

	if len(item.Key) > infohash.Size {
		http.Error(rw, fmt.Sprintf("key is too long: %d > %d", len(item.Key), infohash.Size), http.StatusBadRequest)
		return
	}

	var keyBytes [infohash.Size]byte
	copy(keyBytes[:], item.Key)

	// check if already exists locally
	_, err = h.vs.Get(keyBytes)
	if err == nil {
		http.Error(rw, fmt.Sprintf("already exists: %s", item.Key), http.StatusForbidden)
		return
	}

	// check if already exists externally
	if len(h.s.PeerStore().GetPeers(keyBytes)) > 0 {
		http.Error(rw, fmt.Sprintf("already exists: %s", item.Key), http.StatusForbidden)
		return
	}

	// save locally
	storeItem := bep44.Item{Key: &keyBytes, V: item.Value}
	err = h.vs.Put(&storeItem)
	if err != nil {
		return
	}

	// announce to everyone
	res, err := h.s.AnnounceTraversal(keyBytes, dht.AnnouncePeer(dht.AnnouncePeerOpts{
		Port: int(netip.MustParseAddrPort(h.s.Addr().String()).Port()),
	}))

	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
	}

	// wait for announce to finish
	<-res.Finished()

	rw.WriteHeader(http.StatusOK)
}

func firstZero(b []byte) int {
	for i, v := range b {
		if v == 0 {
			return i
		}
	}
	return len(b)
}

func (h *HTTPHandler) GetMany(rw http.ResponseWriter, r *http.Request) {
	u, err := url.Parse(r.URL.String())
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	query, err := url.ParseQuery(u.RawQuery)
	prefix := query.Get("prefix")
	var items []Item

	// lookup locally
	for key, item := range h.vs.GetAll() {
		keyStr := string(key[:firstZero(key[:])])
		if strings.HasPrefix(keyStr, prefix) {
			items = append(items, Item{Key: keyStr, Value: item.V.(string)})
		}
	}

	// lookup externally
	for key, peers := range h.ps.GetAll() {
		keyStr := string(key[:firstZero(key[:])])
		if strings.HasPrefix(keyStr, prefix) && len(peers) > 0 {
			res := h.s.Get(context.TODO(), dht.NewAddr(peers[0].NodeAddr.UDP()), key, nil, dht.QueryRateLimiting{})
			if res.ToError() != nil {
				http.Error(rw, res.ToError().Error(), http.StatusInternalServerError)
				return
			}
			if (res.Reply.R == nil) || (res.Reply.R.V == nil) {
				continue
			}
			bytes, err := res.Reply.R.V.MarshalBencode()
			if err != nil {
				http.Error(rw, err.Error(), http.StatusInternalServerError)
				return
			}
			items = append(items, Item{Key: keyStr, Value: string(bytes)})
		}
	}

	rawResponse, _ := json.Marshal(items)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write(rawResponse)
}

func (h *HTTPHandler) DhtDebug(rw http.ResponseWriter, _ *http.Request) {
	h.s.WriteStatus(rw)
}

func (h *HTTPHandler) DhtPing(rw http.ResponseWriter, _ *http.Request) {
	var resps []string
	for _, node := range h.s.Nodes() {
		res := h.s.Ping(node.Addr.UDP())
		err := res.Err
		if err != nil {
			resps = append(resps, fmt.Sprintf("%s: %s", node.Addr.String(), err.Error()))
		}
		resps = append(resps, fmt.Sprintf("%s: %x", node.Addr.String(), *res.Reply.SenderID()))
	}

	rawResponse, _ := json.Marshal(resps)
	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)
	_, _ = rw.Write(rawResponse)
}

type DhtGetResp struct {
	ID    string        `json:"id"`
	IP    string        `json:"ip"`
	Key   [32]byte      `json:"key"`
	Value bencode.Bytes `json:"value"`
}

func (h *HTTPHandler) DhtGet(rw http.ResponseWriter, r *http.Request) {
	key := strings.Split(r.URL.Path, "/")[3]
	if len(key) > infohash.Size {
		http.Error(rw, fmt.Sprintf("key is too long: %d > %d", len(key), infohash.Size), http.StatusBadRequest)
		return
	}

	var keyBytes [infohash.Size]byte
	copy(keyBytes[:], key)

	if h.s.PeerStore() == nil {
		panic("no peer store")
	}
	peers := h.s.PeerStore().GetPeers(keyBytes)
	if len(peers) == 0 {
		http.Error(rw, fmt.Sprintf("no peers for %s", key), http.StatusNotFound)
		return
	}

	res := h.s.Get(context.TODO(), dht.NewAddr(h.s.Nodes()[0].Addr.UDP()), keyBytes, nil, dht.QueryRateLimiting{})
	err := res.ToError()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	response := DhtGetResp{IP: res.Reply.IP.String(), ID: res.Reply.SenderID().String(), Key: res.Reply.R.K, Value: res.Reply.R.V}
	rawResponse, _ := json.Marshal(response)
	rw.WriteHeader(http.StatusBadRequest)
	_, _ = rw.Write(rawResponse)
}

func NewServer(s *dht.Server, vs *store.Memory, ps *peer_store.InMemory) *http.Server {
	r := mux.NewRouter()

	handler := NewHTTPHandler(s, vs, ps)

	// api
	r.HandleFunc("/api/v1/put", handler.Put).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/getMany", handler.GetMany).Methods(http.MethodGet)

	// dht debug
	r.HandleFunc("/dht/debug", handler.DhtDebug).Methods(http.MethodGet)
	r.HandleFunc("/dht/ping", handler.DhtPing).Methods(http.MethodGet)
	r.HandleFunc("/dht/get/{key}", handler.DhtGet).Methods(http.MethodGet)

	srv := &http.Server{
		Handler:      r,
		Addr:         fmt.Sprintf("0.0.0.0:%s", os.Getenv("SERVER_PORT")),
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}
	return srv
}
