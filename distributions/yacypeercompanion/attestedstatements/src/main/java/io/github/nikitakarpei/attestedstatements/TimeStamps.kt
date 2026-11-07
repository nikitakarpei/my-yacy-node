// Copied from attested-web-capture-android 0.2.0 (commit 75cc4a5); see its design notes there.
package io.github.nikitakarpei.attestedstatements

import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import org.bouncycastle.cms.jcajce.JcaSimpleSignerInfoVerifierBuilder
import org.bouncycastle.jce.provider.BouncyCastleProvider
import org.bouncycastle.tsp.TSPAlgorithms
import org.bouncycastle.tsp.TSPValidationException
import org.bouncycastle.tsp.TimeStampRequest
import org.bouncycastle.tsp.TimeStampRequestGenerator
import org.bouncycastle.tsp.TimeStampResponse
import java.math.BigInteger
import java.security.SecureRandom
import java.security.cert.CertificateFactory
import java.security.cert.X509Certificate
import java.util.concurrent.CompletableFuture
import java.util.concurrent.LinkedBlockingQueue
import java.util.concurrent.TimeUnit
import kotlin.concurrent.thread

const val TIMESTAMP_WAIT_MS = 60_000L

const val TSA_URL = "https://freetsa.org/tsr"

const val MAX_SIGNATURES_PER_TIMESTAMP = 32

/** FreeTSA's signing certificate (https://freetsa.org/files/tsa.crt, valid 2026–2040), SHA-256
 *  32:E8:41:A9:5C:C1:16:41:01:FF:DE:41:29:8E:F2:FC:75:C1:C4:37:2E:F0:95:E8:8A:6B:BD:47:DF:B1:91:FC.
 *  Pinned rather than requested with each token, which saves ~3.7 KB per capture. */
const val TSA_CERT_PEM = """-----BEGIN CERTIFICATE-----
MIIGYDCCBEigAwIBAgIJAMLphhYNqOnNMA0GCSqGSIb3DQEBDQUAMIGVMREwDwYD
VQQKEwhGcmVlIFRTQTEQMA4GA1UECxMHUm9vdCBDQTEYMBYGA1UEAxMPd3d3LmZy
ZWV0c2Eub3JnMSIwIAYJKoZIhvcNAQkBFhNidXNpbGV6YXNAZ21haWwuY29tMRIw
EAYDVQQHEwlXdWVyemJ1cmcxDzANBgNVBAgTBkJheWVybjELMAkGA1UEBhMCREUw
HhcNMjYwMjE1MTk0NDIyWhcNNDAwMjAyMTk0NDIyWjCCAQsxETAPBgNVBAoMCEZy
ZWUgVFNBMQwwCgYDVQQLDANUU0ExdjB0BgNVBA0MbVRoaXMgY2VydGlmaWNhdGUg
ZGlnaXRhbGx5IHNpZ25zIGRvY3VtZW50cyBhbmQgdGltZSBzdGFtcCByZXF1ZXN0
cyBtYWRlIHVzaW5nIHRoZSBmcmVldHNhLm9yZyBvbmxpbmUgc2VydmljZXMxGDAW
BgNVBAMMD3d3dy5mcmVldHNhLm9yZzEkMCIGCSqGSIb3DQEJARYVYnVzaWxlemFz
QG1haWxib3gub3JnMRIwEAYDVQQHDAlXdWVyemJ1cmcxCzAJBgNVBAYTAkRFMQ8w
DQYDVQQIDAZCYXllcm4wdjAQBgcqhkjOPQIBBgUrgQQAIgNiAASiFeGhstbLhxix
0o4UAumNSwHUUlOe3DBvs8fYs580wADW59oqGSCx15bp61TSmXkwLm1JW48XnbLL
izP6ZtjcvshV3H9uz2bS53sgDXhg1wLbIhAtraC+fHCytHeuVaujggHmMIIB4jAJ
BgNVHRMEAjAAMB0GA1UdDgQWBBQVwL0m69RdgtFdkyYxL+9wsotGXjAfBgNVHSME
GDAWgBT6VQ2MNGZRQ0z357OnbJWveuaklzALBgNVHQ8EBAMCBsAwFgYDVR0lAQH/
BAwwCgYIKwYBBQUHAwgwbAYIKwYBBQUHAQEEYDBeMDMGCCsGAQUFBzAChidodHRw
Oi8vd3d3LmZyZWV0c2Eub3JnL2ZpbGVzL2NhY2VydC5wZW0wJwYIKwYBBQUHMAGG
G2h0dHA6Ly93d3cuZnJlZXRzYS5vcmc6MjU2MDA3BgNVHR8EMDAuMCygKqAohiZo
dHRwOi8vd3d3LmZyZWV0c2Eub3JnL2NybC9yb290X2NhLmNybDCByAYDVR0gBIHA
MIG9MIG6BgMrBQgwgbIwMwYIKwYBBQUHAgEWJ2h0dHA6Ly93d3cuZnJlZXRzYS5v
cmcvZnJlZXRzYV9jcHMuaHRtbDAyBggrBgEFBQcCARYmaHR0cDovL3d3dy5mcmVl
dHNhLm9yZy9mcmVldHNhX2Nwcy5wZGYwRwYIKwYBBQUHAgIwOxo5RnJlZVRTQSB0
cnVzdGVkIHRpbWVzdGFtcGluZyBTb2Z0d2FyZSBhcyBhIFNlcnZpY2UgKFNhYVMp
MA0GCSqGSIb3DQEBDQUAA4ICAQBrMVS/YfnfMr0ziZnesBUOrDNRrNNgt3IgMNDw
Nhwl6oKWHVIhlYnM/5boljfbpZTAbqvxHI3ztT0/swxQOqTat5qBJRAY/VH1n/T4
M9uDjSuu3qfh0ZH5PL9ENqoVW44i5NT/znQev2MGXOAHwz9kZwwzz9MFX6hbGhBq
Wa+nlAqb7Y72KFzj33m1OVHxV2Wl4YD9f91bZTFpUEGW4Ktbkmxpf/iGIPaf4WHp
oBW/O6EzofMKYlz4yXyEBh0wRRVyXltLrj+MFHqhe+PsMBllq/dCaO4W/F+AuHEl
u7aUYWMASelphWAJiUsNMr5HAoeCSSgilqf1CSoWC+k6e4334Fym+Iy4csMex+PG
4rSdqXJVQ+AWEdRajSPKh7yDfpNkdnO6yqQJ/tSd11XQ5cL0M9jWuCD1zHlgA+u+
R2cry3yo23jD7qTGLhZqUvXCyWigH30/Q/RXjjDwrc4DJiQ+gRY0FhdTYqlvgMBP
r4LcJKnNksivdj+kbz7bVSbrBAzRiazK9l841/5XMtP9BvD0hKCpQFvP9PSgCC8E
QnKqgSe26FSJBaAQcA5TnK8NF4jkbElBxf/zyh7P3IjHso35jtgUWD1/itg9BJWb
YUwJ4tfILpB2F0wbk1GcZDCDZoyW3Xf3trApz/Zd93gF3joc9Hh9RFveKRzWQ7dd
Ut3egQ==
-----END CERTIFICATE-----"""

val BC by lazy { BouncyCastleProvider() } // not registered system-wide, so it can't replace Android's crypto

val TSA_CERT: X509Certificate by lazy {
    CertificateFactory.getInstance("X.509").generateCertificate(TSA_CERT_PEM.byteInputStream()) as X509Certificate
}

/** Returns the TSA's DER TimeStampResp for SHA-256 [digest]. */
fun timestamp(client: OkHttpClient, digest: ByteArray): ByteArray {
    val request = TimeStampRequestGenerator().generate(TSPAlgorithms.SHA256, digest, BigInteger(64, SecureRandom()))
    val bytes = client.newCall(
        Request.Builder().url(TSA_URL)
            .post(request.encoded.toRequestBody("application/timestamp-query".toMediaType())).build(),
    ).execute()
        .use { r ->
            check(r.code == 200) { "timestamp authority returned HTTP ${r.code}" }
            r.body.bytes()
        }
    checkTimestamp(request, bytes)
    return bytes
}

/** The requester's checks from RFC 3161 section 2.2, except revocation (a SHOULD), which verifiers can do. */
fun checkTimestamp(request: TimeStampRequest, bytes: ByteArray) {
    val response = TimeStampResponse(bytes)
    check(response.status == 0) { "timestamp not granted: ${response.statusString}" }
    response.validate(request) // hash algorithm, imprint, nonce
    val token = checkNotNull(response.timeStampToken) { "no time-stamp token in response" }
    // Also checks the certificate's timeStamping usage and validity.
    try {
        token.validate(JcaSimpleSignerInfoVerifierBuilder().setProvider(BC).build(TSA_CERT))
    } catch (e: TSPValidationException) {
        throw IllegalStateException("${e.message} (if FreeTSA has a new signing certificate, the app needs an update)", e)
    }
}

/** FreeTSA allows one request per second per IP address, so captures waiting at the same time share a
 *  time-stamp over the root of a Merkle tree of their signed-statement IDs; each response carries its path. */
class Stamper(private val client: OkHttpClient) : AutoCloseable {
    /** [path] proves the signed statement is leaf [index] of the [size]-leaf tree whose root [token] time-stamps. */
    class Stamp(val token: ByteArray, val size: Int, val index: Int, val path: List<ByteArray>)
    private class Pending(val signedId: ByteArray, val result: CompletableFuture<Stamp> = CompletableFuture())

    private val queue = LinkedBlockingQueue<Pending>()
    private val worker = thread(isDaemon = true, name = "time-stamps") {
        try {
            while (true) {
                val batch = mutableListOf(queue.take())
                queue.drainTo(batch, MAX_SIGNATURES_PER_TIMESTAMP - 1)
                val leaves = batch.map { merkleLeaf(it.signedId) }
                try {
                    val token = timestamp(client, merkleRoot(leaves))
                    batch.forEachIndexed { i, p -> p.result.complete(Stamp(token, leaves.size, i, merklePath(i, leaves))) }
                } catch (e: Exception) {
                    batch.forEach { it.result.completeExceptionally(e) }
                }
                Thread.sleep(1_000)
            }
        } catch (e: InterruptedException) {
            // closed
        }
    }

    fun stamp(signedId: ByteArray): Stamp = Pending(signedId).also { queue.put(it) }.result.get(TIMESTAMP_WAIT_MS, TimeUnit.MILLISECONDS)

    override fun close() = worker.interrupt()
}

fun merkleLeaf(data: ByteArray) = sha256(byteArrayOf(0) + data)

fun merkleRoot(leaves: List<ByteArray>): ByteArray {
    if (leaves.size == 1) return leaves[0]
    val k = Integer.highestOneBit(leaves.size - 1) // the largest power of two smaller than the size
    return sha256(byteArrayOf(1) + merkleRoot(leaves.subList(0, k)) + merkleRoot(leaves.subList(k, leaves.size)))
}

/** The inclusion proof of leaf [m], nearest sibling first. */
fun merklePath(m: Int, leaves: List<ByteArray>): List<ByteArray> {
    if (leaves.size == 1) return emptyList()
    val k = Integer.highestOneBit(leaves.size - 1)
    return if (m < k) {
        merklePath(m, leaves.subList(0, k)) + listOf(merkleRoot(leaves.subList(k, leaves.size)))
    } else {
        merklePath(m - k, leaves.subList(k, leaves.size)) + listOf(merkleRoot(leaves.subList(0, k)))
    }
}
