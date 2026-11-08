package io.github.nikitakarpei.yacypeercompanion.services

import android.content.Context
import android.util.Log
import io.github.nikitakarpei.httpcapture.CaptureProxy
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent
import io.github.nikitakarpei.yacypeercompanion.model.ServicesEnding
import pageservices.Instance
import pageservices.Observer
import pageservices.Pageservices
import kotlin.concurrent.thread

private const val LOG_TAG = "ServicesRunner"

class ServicesRunner(private val ctx: Context, private val events: (CompanionEvent) -> Unit) : AutoCloseable {
    private var proxy: CaptureProxy? = null
    private var services: Instance? = null

    @Volatile
    private var currentRun = 0

    fun start(linkText: String) {
        stop()
        val run = ++currentRun
        val proxy = proxy ?: openProxy() ?: return events(CompanionEvent.SigningKeyUnavailable)
        val key = DeviceKeyEntry.current() ?: return events(CompanionEvent.DeviceKeyUnavailable)
        val instance =
            runCatching { Pageservices.instanceFor(linkText, proxy.url, key, appVersionOf(ctx)) }.getOrElse {
                Log.w(LOG_TAG, "page services", it)
                return events(CompanionEvent.ServicesEnded(ServicesEnding.FAILED))
            }
        services = instance
        thread(name = "page-services-$run") { instance.run(observerFor(run)) }
    }

    fun stop() {
        currentRun++
        services?.stop()
        services = null
    }

    private fun openProxy(): CaptureProxy? = runCatching { CaptureProxy.open(ctx) }
        .onFailure { Log.w(LOG_TAG, "signing key", it) }
        .getOrNull()
        ?.also { proxy = it }

    private fun observerFor(run: Int) = object : Observer {
        override fun servicesConnected() = report(run, CompanionEvent.ServicesConnected)

        override fun brokerConnectionLost() = report(run, CompanionEvent.BrokerConnectionLost)

        override fun servicesStopped() = report(run, CompanionEvent.ServicesEnded(ServicesEnding.STOPPED))

        override fun brokerUnreachable() = report(run, CompanionEvent.ServicesEnded(ServicesEnding.BROKER_UNREACHABLE))

        override fun servicesFailed() = report(run, CompanionEvent.ServicesEnded(ServicesEnding.FAILED))

        override fun deviceRefused() = report(run, CompanionEvent.ServicesEnded(ServicesEnding.DEVICE_REFUSED))
    }

    private fun report(run: Int, event: CompanionEvent) {
        if (run == currentRun) events(event)
    }

    override fun close() {
        stop()
        proxy?.close()
        proxy = null
    }
}
