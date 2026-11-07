// Copied from attested-web-capture-android 0.2.0 (commit 75cc4a5); see its design notes there.
package io.github.nikitakarpei.attestedstatements

import okhttp3.OkHttpClient
import okhttp3.Request
import org.json.JSONObject

const val DRAND_CHAIN = "52db9ba70e0cc0f6eaf7803dd07447a1f5477735fd3f661792ba94600c84e971" // quicknet: a round every 3 s

val DRAND_MIRRORS = listOf("https://api.drand.sh", "https://api2.drand.sh", "https://drand.cloudflare.com")

class Beacon(val round: Long, val signature: String)

/** Fetched on demand, at most once per round, so idle phones don't poll. Not verified here: a bad value only
 *  makes the capture fail verification, and the verifier checks it. */
class Drand(private val client: OkHttpClient, private val clock: () -> Long) {
    private var cached: Pair<Long, Beacon>? = null

    @Synchronized fun latest(): Beacon {
        cached?.let { (fetchedAt, beacon) -> if (clock() - fetchedAt < 3_000) return beacon }
        var failure: Exception? = null
        for (mirror in DRAND_MIRRORS) {
            try {
                val beacon = fetch("$mirror/$DRAND_CHAIN/public/latest")
                cached = clock() to beacon
                return beacon
            } catch (e: Exception) {
                failure = e
            }
        }
        throw Retryable("drand unavailable: ${failure?.message}", failure)
    }

    private fun fetch(url: String): Beacon = client.newCall(Request.Builder().url(url).build()).execute().use { r ->
        check(r.code == 200) { "drand returned HTTP ${r.code}" }
        val json = JSONObject(r.body.string())
        val signature = json.getString("signature")
        check(signature.length == 96 && signature.all { it in "0123456789abcdef" }) { "malformed drand signature" }
        Beacon(json.getLong("round"), signature)
    }
}
