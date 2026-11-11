package io.github.nikitakarpei.yacypeercompanion.services

import io.github.nikitakarpei.attestedstatements.Stamper.Stamp
import org.json.JSONArray
import org.json.JSONObject
import java.util.Base64

fun Stamp.encoded(): ByteArray {
    val encoder = Base64.getEncoder()
    return JSONObject()
        .put("token", encoder.encodeToString(token))
        .put(
            "tree",
            JSONObject()
                .put("size", size)
                .put("index", index)
                .put("path", JSONArray(path.map { encoder.encodeToString(it) })),
        )
        .toString()
        .toByteArray()
}
