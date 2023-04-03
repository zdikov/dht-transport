# Транспорт поверх DHT

## Демо: [dht-transport.mp4](https://disk.yandex.ru/i/GArlgdCbAfKVEA)

Проект состоит из сервера и клиентской библотеки, которая общается с сервером по HTTP.

## Интерфейс библиотеки
```kotlin
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
```

## Описание сервера: [dht/README.md](dht/README.md)
