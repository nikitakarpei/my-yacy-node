package io.github.nikitakarpei.yacypeercompanion.services

import io.github.nikitakarpei.attestedstatements.SigningKey
import io.github.nikitakarpei.attestedstatements.Stamper
import io.github.nikitakarpei.attestedstatements.keyEvidence
import org.json.JSONArray
import org.json.JSONObject
import pageservices.HardwareKey
import pageservices.HardwareKeys
import java.util.Base64

class StatementKeys(private val signingKey: () -> SigningKey, private val stamper: Stamper) : HardwareKeys {
    override fun currentKey(): HardwareKey = HardwareStatementKey(signingKey())

    override fun timeStampOf(signedStatementID: ByteArray): ByteArray {
        val stamp = stamper.stamp(signedStatementID)
        val encoder = Base64.getEncoder()
        return JSONObject()
            .put("token", encoder.encodeToString(stamp.token))
            .put(
                "tree",
                JSONObject()
                    .put("size", stamp.size)
                    .put("index", stamp.index)
                    .put("path", JSONArray(stamp.path.map { encoder.encodeToString(it) })),
            )
            .toString()
            .toByteArray()
    }
}

private class HardwareStatementKey(private val key: SigningKey) : HardwareKey {
    override fun keyEvidence(): ByteArray = keyEvidence(key.chain)

    override fun publicKeyInfo(): ByteArray = key.chain[0].publicKey.encoded

    override fun sign(message: ByteArray): ByteArray = key.sign(message)
}
