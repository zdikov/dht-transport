package dht_transport

import kotlinx.serialization.Serializable

@Serializable
data class Message(
    val messageId: ULong,
    val parentId: ULong,
    val userId: String,
    val text: String,
    val time: String,
) {}