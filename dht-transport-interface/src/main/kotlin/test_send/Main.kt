package test_send

import dht_transport.Message
import dht_transport.DHTTransport
import com.github.ajalt.clikt.core.CliktCommand
import com.github.ajalt.clikt.parameters.options.*
import java.time.LocalDateTime
import java.time.format.DateTimeFormatter

class Send : CliktCommand() {
    val dht_address: String by option(help="DHT address").prompt("Enter dht_address")
    val channel_id: String by option(help="Channel ID").prompt("Enter channel_id")
    val user_id: String by option(help="User ID").prompt("Enter user_id")

    override fun run() {
        val p = DHTTransport(dht_address)
        var id = 1UL
        val formatter = DateTimeFormatter.ofPattern("yyyy-MM-dd HH:mm")
        while (true) {
            val text = prompt("Type text")!!
            val parent_id = prompt("Type parent_id")!!.toULong()
            val current = LocalDateTime.now()
            val formatted = current.format(formatter)
            val msg = Message(id, parent_id, user_id, text, formatted)
            p.sendMessages(channel_id, arrayOf<Message>(msg))
            println("Text sent with id $id")
            id = id.inc()
        }
    }
}

fun main(args: Array<String>) = Send().main(args)
