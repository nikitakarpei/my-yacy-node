// Copied from attested-web-capture-android 0.2.0 (commit 75cc4a5); see its design notes there.
package io.github.nikitakarpei.httpcapture

import io.github.nikitakarpei.attestedstatements.Drand
import io.github.nikitakarpei.attestedstatements.Retryable
import io.github.nikitakarpei.attestedstatements.SigningKey
import io.github.nikitakarpei.attestedstatements.Stamper
import io.github.nikitakarpei.attestedstatements.keyEvidence
import okhttp3.HttpUrl.Companion.toHttpUrlOrNull
import okhttp3.OkHttpClient
import java.io.BufferedInputStream
import java.io.ByteArrayOutputStream
import java.io.IOException
import java.io.InputStream
import java.net.InetAddress
import java.net.Proxy
import java.net.ServerSocket
import java.net.Socket
import java.security.MessageDigest
import java.security.Signature
import java.util.Base64
import java.util.concurrent.ArrayBlockingQueue
import java.util.concurrent.RejectedExecutionException
import java.util.concurrent.ThreadPoolExecutor
import java.util.concurrent.TimeUnit
import kotlin.concurrent.thread

const val VERSION = "0.6.0"

const val ATTESTED_VERSION = "v0.1"

const val REQUEST_READ_DEADLINE_MS = 10_000L

const val MAX_CONCURRENT_CAPTURES = 8

const val MAX_QUEUED_CONNECTIONS = 64

const val RETRY_AFTER_SECONDS = 10

private const val LOG_TAG = "CaptureProxy"

fun statusFor(e: Exception) = when (e) {
    is IllegalArgumentException, is Refused -> 400
    is Retryable -> 503
    else -> 502
}

val HOP_BY_HOP = setOf(
    "connection", "keep-alive", "proxy-connection", "proxy-authenticate",
    "transfer-encoding", "te", "trailer", "upgrade", "content-length",
)

/** Origin's Attested-* headers are dropped (they stay in the statement), so it can't pass off its own. */
fun forwardedHeaders(headers: List<Pair<String, String>>) = headers.filter { (name, _) ->
    val n = name.lowercase()
    n !in HOP_BY_HOP && !n.startsWith("attested-")
}

fun serve(
    server: ServerSocket,
    token: String,
    key: () -> SigningKey,
    client: OkHttpClient,
    acceptOn: (InetAddress) -> Boolean = { LOCAL_NETWORKS.matches(it) },
    clock: () -> Long,
) {
    // Bounds memory (each capture may hold a full body); connections beyond the queue are closed.
    val pool = ThreadPoolExecutor(
        MAX_CONCURRENT_CAPTURES,
        MAX_CONCURRENT_CAPTURES,
        0,
        TimeUnit.MILLISECONDS,
        ArrayBlockingQueue(MAX_QUEUED_CONNECTIONS),
    )
    val stamper = Stamper(client)
    val drand = Drand(client, clock)
    try {
        while (true) {
            val socket = try {
                server.accept()
            } catch (e: IOException) {
                break // closed
            }
            if (!acceptOn(socket.localAddress)) {
                socket.close()
                continue
            }
            try {
                pool.execute { socket.use { handle(it, token, key, client, stamper, drand) } }
            } catch (e: RejectedExecutionException) {
                socket.close()
            }
        }
    } finally {
        pool.shutdownNow()
        stamper.close()
    }
}

fun handle(socket: Socket, token: String, key: () -> SigningKey, client: OkHttpClient, stamper: Stamper, drand: Drand) {
    socket.soTimeout = 5_000
    val out = socket.getOutputStream()
    fun send(status: Int, reason: String, headers: List<Pair<String, String>>, body: ByteArray) {
        val head = StringBuilder("HTTP/1.1 $status $reason\r\n")
        for ((name, value) in headers) head.append(name).append(": ").append(value).append("\r\n")
        head.append("Content-Length: ${body.size}\r\nConnection: close\r\n\r\n")
        out.write(head.toString().toByteArray(Charsets.ISO_8859_1))
        out.write(body)
        out.flush()
    }
    fun error(status: Int, message: String, extra: List<Pair<String, String>> = emptyList()) =
        send(status, "Error", listOf("Content-Type" to "text/plain; charset=utf-8") + extra, "$message\n".toByteArray())
    try {
        // A deadline for the whole request, not just per read, so slow senders can't hold a thread.
        val deadline = System.nanoTime() + REQUEST_READ_DEADLINE_MS * 1_000_000
        val input = BufferedInputStream(socket.getInputStream())
        val request = readLine(input, deadline).split(' ')
        val headers = generateSequence { readLine(input, deadline).takeIf { it.isNotEmpty() } }.take(100).toList()
        fun header(name: String) = headers.firstOrNull { it.startsWith("$name:", ignoreCase = true) }
            ?.substringAfter(':')?.trim() ?: ""
        if (!authorized(header("proxy-authorization"), token)) {
            return error(
                407,
                "proxy authentication required: use the token as the proxy password",
                listOf("Proxy-Authenticate" to "Basic realm=\"attested-web-capture\""),
            )
        }
        if (request.size != 3) return error(400, "malformed request line")
        val (method, target) = request
        if (method == "CONNECT") return error(405, "CONNECT is not supported: request http://… URLs instead; they are fetched over HTTPS")
        if (method != "GET") return error(405, "only GET is supported")
        val url = target.toHttpUrlOrNull()
            ?: return error(400, "send absolute http:// URLs (use this server as an HTTP proxy)")
        if (url.port != 80 && url.port != 443) return error(400, "only the default ports are supported")
        // Required, because it's sent and signed: the record shows which client the page was served to.
        val userAgent = header("user-agent")
        if (userAgent.isEmpty() || userAgent.length > 512 || userAgent.any { it < ' ' || it > '~' }) {
            return error(
                400,
                "a User-Agent header is required (printable ASCII, at most 512 characters); " +
                    "it is sent to the target and recorded in the signed statement",
            )
        }
        val https = url.newBuilder().scheme("https").port(443).build()
        val k = try {
            key()
        } catch (e: Exception) {
            throw Retryable("signing key unavailable: ${e.message}", e)
        }
        val r = capture(https, userAgent, k, drand.latest(), client, stamper)
        send(
            r.status,
            r.reason,
            forwardedHeaders(r.headers) + listOf(
                "Attested-Version" to ATTESTED_VERSION,
                "Attested-Statement" to sfBytes(r.statement),
                "Attested-Signature" to sfBytes(r.signature),
                "Attested-Key" to sfBytes(keyEvidence(r.chain)),
                "Attested-Timestamp" to sfBytes(r.stamp.token),
                "Attested-Timestamp-Tree" to
                    "size=${r.stamp.size}, index=${r.stamp.index}, path=(${r.stamp.path.joinToString(" ") { sfBytes(it) }})",
            ),
            r.body,
        )
    } catch (e: Exception) {
        android.util.Log.w(LOG_TAG, "capture failed", e)
        val retry = if (e is Retryable) listOf("Retry-After" to "$RETRY_AFTER_SECONDS") else emptyList()
        runCatching { error(statusFor(e), "capture failed: ${e.message}", retry) }
    }
}

/** "Proxy-Authorization: Basic base64(anything:token)", as curl -x http://:token@host sends it. */
fun authorized(value: String, token: String): Boolean {
    if (!value.startsWith("Basic ", ignoreCase = true)) return false
    val given = runCatching { String(Base64.getDecoder().decode(value.substring(6).trim())).substringAfter(':') }
        .getOrDefault("")
    return MessageDigest.isEqual(given.toByteArray(), token.toByteArray())
}

/** [deadline]: System.nanoTime(). */
fun readLine(input: InputStream, deadline: Long): String {
    val b = ByteArrayOutputStream()
    while (true) {
        require(System.nanoTime() < deadline) { "request not received in time" }
        val c = input.read()
        if (c < 0 || c == '\n'.code) break
        if (c != '\r'.code) b.write(c)
        require(b.size() <= 8192) { "request line too long" }
    }
    return b.toString("ISO-8859-1")
}

fun sfBytes(b: ByteArray) = ":" + Base64.getEncoder().encodeToString(b) + ":" // RFC 9651 byte sequence
