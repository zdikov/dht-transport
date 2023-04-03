package test_receive

import dht_transport.Message
import dht_transport.DHTTransport
import com.github.ajalt.clikt.core.CliktCommand
import com.github.ajalt.clikt.parameters.options.*

class Receive : CliktCommand() {
    val dht_address: String by option(help="DHT address").prompt("Enter dht_address")
    val channel_id: String by option(help="Channel ID").prompt("Enter channel_id")

    override fun run() {
        val p = DHTTransport(dht_address)
        while (true) {
            val parent_id = prompt("Type parent_id")!!.toULong()
            val result: Array<Message> = p.receiveMessages(channel_id, parent_id)
            if (result.isEmpty()) {
                println("No messages")
                continue
            }
            for (msg in result) {
                println(msg.time + ' ' + msg.userId + ": " + msg.text)
            }
        }
    }
}

fun main(args: Array<String>) = Receive().main(args)
