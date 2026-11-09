package io.github.nikitakarpei.httpcapture

import android.content.Context
import android.content.pm.ApplicationInfo
import android.os.SystemClock
import io.github.nikitakarpei.attestedstatements.Keys
import io.github.nikitakarpei.attestedstatements.SigningKey
import io.github.nikitakarpei.attestedstatements.hex
import java.net.InetAddress
import java.net.Proxy
import java.net.ServerSocket
import java.security.SecureRandom
import kotlin.concurrent.thread

/**
 * The capture proxy on a free loopback port, for the pagescrape of this app only. It asks
 * for the password in [url] and signs each response with a fresh hardware-attested key.
 */
private const val LOOPBACK_HOST = "127.0.0.1"

class CaptureProxy private constructor(private val server: ServerSocket, private val password: String) : AutoCloseable {
    val url: String get() = "http://:$password@$LOOPBACK_HOST:${server.localPort}"

    override fun close() {
        runCatching { server.close() }
    }

    companion object {
        /** A debuggable build can be made to sign anything, so it never gets a key (captures answer 503). */
        fun open(ctx: Context): CaptureProxy {
            val debuggable = ctx.applicationInfo.flags and ApplicationInfo.FLAG_DEBUGGABLE != 0
            Keys.reset()
            if (!debuggable) Keys.get()
            val key: () -> SigningKey = if (debuggable) { -> error("debuggable build: refusing to sign") } else Keys::get
            val server = ServerSocket(0, MAX_QUEUED_CONNECTIONS, InetAddress.getByName(LOOPBACK_HOST))
            val password = hex(ByteArray(16).also { SecureRandom().nextBytes(it) })
            thread(name = "capture-proxy") {
                serve(server, password, key, httpClient(Proxy.NO_PROXY), { it.isLoopbackAddress }, SystemClock::elapsedRealtime)
            }
            return CaptureProxy(server, password)
        }
    }
}
