package io.github.nikitakarpei.yacypeercompanion.services

import io.github.nikitakarpei.attestedstatements.Stamper
import org.json.JSONArray
import org.json.JSONObject
import pageservices.TimeStampAuthority
import java.util.Base64

class StatementTimeStampAuthority(private val stamper: Stamper) : TimeStampAuthority {
    override fun stamp(signedStatementID: ByteArray): ByteArray {
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
