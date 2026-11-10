package io.github.nikitakarpei.yacypeercompanion.services

import io.github.nikitakarpei.attestedstatements.SigningKey
import io.github.nikitakarpei.attestedstatements.keyEvidence
import pageservices.HardwareKey
import pageservices.HardwareKeys

class StatementKeys(private val signingKey: () -> SigningKey) : HardwareKeys {
    override fun currentKey(): HardwareKey = HardwareStatementKey(signingKey())
}

private class HardwareStatementKey(private val key: SigningKey) : HardwareKey {
    override fun keyEvidence(): ByteArray = keyEvidence(key.chain)

    override fun publicKeyInfo(): ByteArray = key.chain[0].publicKey.encoded

    override fun sign(message: ByteArray): ByteArray = key.sign(message)
}
