package dht_transport

import java.net.http.HttpClient
import java.net.http.HttpRequest
import java.net.http.HttpResponse
import java.net.URI
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.Serializable

@Serializable
data class KeyValuePair(val key: String, val value: String) {}

interface Transport {
    // sends a message to the peer `dest` (channel id);
    // note that the message has an lseq (messageId) inside
    // `dest` — messageId of the first message in channel
    fun sendMessages(dest: String, msgs: Array<Message>)
    // receives new messages for `src` (channel id)
    // last_received is the lseq of the last message
    // received in this channel
    fun receiveMessages(src: String, lastReceived: ULong) : Array<Message>
}

class DHTTransport(val dhtAddress: String) : Transport {
    override fun sendMessages(dest: String, msgs: Array<Message>) {
        val client = HttpClient.newBuilder().build()
        for (msg in msgs) {
            val request = HttpRequest.newBuilder()
                .uri(URI.create("${dhtAddress}/api/v1/put/"))
                .setHeader("Content-Type", "application/json")
                .POST(
                    HttpRequest.BodyPublishers.ofString(
                        Json.encodeToString(
                            KeyValuePair(
                                "${dest}.${msg.parentId}.${msg.messageId}",
                                Json.encodeToString(msg)
                            )
                        )
                    )
                )
                .build()

            val response = client.send(request, HttpResponse.BodyHandlers.ofString())
        }
    }

    override fun receiveMessages(src: String, lastReceived: ULong) : Array<Message> {
        val client = HttpClient.newBuilder().build()
        val request = HttpRequest.newBuilder()
            .uri(URI.create("${dhtAddress}/api/v1/getMany/?prefix=${src}.${lastReceived}."))
            .GET()
            .build()

        val response = client.send(request, HttpResponse.BodyHandlers.ofString())
        val stringBody: String = response.body()

        var result: Array<Message> = arrayOf<Message>()
        var kvs: Array<KeyValuePair> = Json.decodeFromString<Array<KeyValuePair>>(stringBody)
        for (kv in kvs) {
            result += Json.decodeFromString<Message>(kv.value)
        }

        return result
    }
}

fun main() {
    // Tests
    println("Enter DHT address. For example, http://127.0.0.1:80")
    var p = DHTTransport(readLine().toString())
    var msg1 = Message(1U, 0U, "nnv-nick", "Hello everyone!", "1:25")
    var msg2 = Message(2U, 1U, "nnv-nick", "Where are you now?", "1:25")
    var msg3 = Message(3U, 2U, "nnv-nick", "No one is here?", "1:30")
    var msg4 = Message(4U, 3U, "timrealdeal", "I'm here, but i don't wanna chat now", "1:31")
    var msg5 = Message(5U, 3U, "zdikov", "I'm here too, what do you need?", "1:31")

    p.sendMessages("1", arrayOf<Message>(msg1, msg2, msg3, msg4, msg5))

    var result: Array<Message> = p.receiveMessages("1", 1U)
    for (msg in result) {
        println(msg)
    }

    result = p.receiveMessages("1", 3U)
    for (msg in result) {
        println(msg)
    }
}