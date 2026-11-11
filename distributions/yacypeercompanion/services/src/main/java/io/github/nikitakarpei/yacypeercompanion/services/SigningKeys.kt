package io.github.nikitakarpei.yacypeercompanion.services

import io.github.nikitakarpei.attestedstatements.SigningKey
import io.github.nikitakarpei.attestedstatements.keyEvidence
import pageservices.HardwareKey

fun SigningKey.asHardwareKey(): HardwareKey {
    val key = this
    return object : HardwareKey {
        override fun keyEvidence(): ByteArray = keyEvidence(key.chain)

        override fun publicKeyInfo(): ByteArray = key.chain[0].publicKey.encoded

        override fun sign(message: ByteArray): ByteArray = key.sign(message)
    }
}
