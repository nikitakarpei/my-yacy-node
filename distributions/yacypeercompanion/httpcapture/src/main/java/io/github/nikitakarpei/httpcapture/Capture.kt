// Copied from attested-web-capture-android 0.2.0 (commit 75cc4a5); see its design notes there.
package io.github.nikitakarpei.httpcapture

import io.github.nikitakarpei.attestedstatements.Beacon
import io.github.nikitakarpei.attestedstatements.DRAND_CHAIN
import io.github.nikitakarpei.attestedstatements.SigningKey
import io.github.nikitakarpei.attestedstatements.hex
import io.github.nikitakarpei.attestedstatements.pae
import io.github.nikitakarpei.attestedstatements.sha256
import okhttp3.ConnectionPool
import okhttp3.CookieJar
import okhttp3.Dns
import okhttp3.HttpUrl
import okhttp3.OkHttpClient
import okhttp3.Protocol
import okhttp3.Request
import okhttp3.ResponseBody
import org.json.JSONArray
import org.json.JSONObject
import java.io.IOException
import java.net.InetAddress
import java.net.InetSocketAddress
import java.net.Proxy
import java.security.cert.X509Certificate
import java.util.concurrent.TimeUnit

const val PREDICATE_TYPE = "https://github.com/nikitakarpei/hardware-attested-statements/blob/main/spec/predicates/http-response/v1.md"

const val MAX_BODY_BYTES = 10 * 1024 * 1024

/** [proxy]: NO_PROXY on the phone, so the system proxy setting can't reroute captures. */
fun httpClient(proxy: Proxy): OkHttpClient = OkHttpClient.Builder()
    .proxy(proxy)
    // OkHttp connects only to the addresses this returns, so a second, different DNS answer can't slip past.
    .dns { host -> Dns.SYSTEM.lookup(host).also { requireAllowed(host, it) } }
    .protocols(listOf(Protocol.HTTP_1_1))
    .cookieJar(CookieJar.NO_COOKIES)
    // Each capture is exactly one request on a fresh connection; redirects go back to the requester.
    .followRedirects(false).followSslRedirects(false)
    .retryOnConnectionFailure(false)
    .connectionPool(ConnectionPool(0, 1, TimeUnit.SECONDS))
    .connectTimeout(15, TimeUnit.SECONDS).readTimeout(30, TimeUnit.SECONDS).callTimeout(50, TimeUnit.SECONDS)
    .build()

class Capture(
    val status: Int,
    val reason: String,
    val headers: List<Pair<String, String>>,
    val body: ByteArray,
    val statement: ByteArray,
    val signature: ByteArray,
    val chain: List<X509Certificate>,
)

/** [startedAfter] is the latest drand round when the capture starts: the statement includes it, which proves the
 *  capture can't have been made before that round's time. */
fun capture(url: HttpUrl, userAgent: String, key: SigningKey, startedAfter: Beacon, client: OkHttpClient): Capture {
    require(url.isHttps && url.port == 443) { "only HTTPS on port 443 is fetched" }
    // OkHttp doesn't look up IP-address hosts, so the DNS filter never sees them.
    requireAllowed(url.host, InetAddress.getAllByName(url.host).toList())

    var connectedTo: InetSocketAddress? = null
    val recording = client.newBuilder().addNetworkInterceptor { chain ->
        connectedTo = chain.connection()!!.route().socketAddress
        chain.proceed(chain.request())
    }.build()
    val request = Request.Builder().url(url).get()
        .header("User-Agent", userAgent)
        .header("Accept-Encoding", "identity") // so the body hash covers the bytes as served
        .build()
    val response = recording.newCall(request).execute()
    val body = response.use { readBody(it.body) }
    val handshake = checkNotNull(response.handshake) { "no TLS handshake" }
    // As sent, including the headers OkHttp adds itself.
    val wire = checkNotNull(response.networkResponse) { "response didn't come from the network" }.request
    val headers = response.headers.map { it.first to it.second }
    val statusLine = "HTTP/1.1 ${response.code} ${response.message}"
    // The body subject names how to read the body; verifiers reject a response with either header twice.
    val contentType = response.headers.values("Content-Type")
    val contentEncoding = response.headers.values("Content-Encoding")
    if (contentType.size > 1 || contentEncoding.size > 1) throw java.io.IOException("the site sent Content-Type or Content-Encoding twice")
    val bodySubject = JSONObject().put("name", "body").put("digest", JSONObject().put("sha256", hex(sha256(body))))
    contentType.firstOrNull()?.let { bodySubject.put("mediaType", it) }
    contentEncoding.firstOrNull()?.let { bodySubject.put("annotations", JSONObject().put("contentEncoding", it)) }

    val predicate = JSONObject()
        .put(
            "request",
            JSONObject()
                .put("method", wire.method)
                .put("url", wire.url.toString())
                .put("headers", JSONArray(wire.headers.map { JSONArray(listOf(it.first, it.second)) }))
                .put("cookies", "none"),
        )
        .put(
            "connection",
            JSONObject()
                .put("route", if (client.proxy == Proxy.NO_PROXY) "direct" else client.proxy.toString())
                .put("connectedTo", connectedTo?.let { "${it.address.hostAddress}:${it.port}" })
                .put("protocol", response.protocol.toString())
                .put("tlsVersion", handshake.tlsVersion.javaName)
                .put("cipherSuite", handshake.cipherSuite.javaName)
                .put("serverCertificateSha256", hex(sha256(handshake.peerCertificates[0].encoded))),
        )
        .put(
            "response",
            JSONObject()
                .put("statusLine", statusLine)
                .put("status", response.code)
                .put("headers", JSONArray(headers.map { JSONArray(listOf(it.first, it.second)) })),
        )
        .put(
            "startedAfter",
            JSONObject().put(
                "drand",
                JSONObject().put("chain", DRAND_CHAIN).put("round", startedAfter.round).put("signature", startedAfter.signature),
            ),
        )
    val statement = JSONObject()
        .put("_type", "https://in-toto.io/Statement/v1")
        .put("subject", JSONArray().put(bodySubject))
        .put("predicateType", PREDICATE_TYPE)
        .put("predicate", predicate)
        .toString().toByteArray()
    val signature = key.sign(pae(statement))
    return Capture(response.code, response.message, headers, body, statement, signature, key.chain)
}

fun readBody(body: ResponseBody): ByteArray {
    val source = body.source()
    check(!source.request(MAX_BODY_BYTES + 1L)) { "response larger than ${MAX_BODY_BYTES / 1024 / 1024} MB" }
    return source.buffer.readByteArray()
}
