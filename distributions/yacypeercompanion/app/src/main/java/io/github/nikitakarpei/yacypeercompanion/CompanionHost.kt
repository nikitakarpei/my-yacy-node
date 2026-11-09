package io.github.nikitakarpei.yacypeercompanion

import android.annotation.SuppressLint
import android.app.NotificationManager
import android.app.Service
import android.content.Intent
import android.os.PowerManager
import android.os.SystemClock
import android.util.Log
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent
import io.github.nikitakarpei.yacypeercompanion.model.CompanionModel
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState
import io.github.nikitakarpei.yacypeercompanion.model.after
import io.github.nikitakarpei.yacypeercompanion.model.keepsDeviceAwake
import io.github.nikitakarpei.yacypeercompanion.services.Pairer
import io.github.nikitakarpei.yacypeercompanion.services.ServicesRunner
import java.util.concurrent.Executors
import java.util.concurrent.LinkedBlockingQueue
import java.util.concurrent.ScheduledFuture
import java.util.concurrent.TimeUnit
import kotlin.concurrent.thread

private const val LOG_TAG = "CompanionHost"

class CompanionHost : Service() {
    private val events = LinkedBlockingQueue<CompanionEvent>()
    private val timers = Executors.newSingleThreadScheduledExecutor()
    private var retry: ScheduledFuture<*>? = null
    private var pairingTimeout: ScheduledFuture<*>? = null
    private lateinit var model: CompanionModel
    private lateinit var services: ServicesRunner
    private lateinit var pairer: Pairer
    private lateinit var network: NetworkWatch
    private lateinit var power: PowerWatch
    private lateinit var wakeLock: PowerManager.WakeLock

    @Volatile
    private var ended = false
    private val wakeLockGuard = Any()

    override fun onBind(intent: Intent?) = null

    override fun onCreate() {
        super.onCreate()
        startForeground(STATUS_NOTIFICATION_ID, statusNotification(this, CompanionStore.state))
        wakeLock = getSystemService(PowerManager::class.java).newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, "$APP_NAME:services")
        services = ServicesRunner(this) { events.put(it) }
        pairer = Pairer(this) { events.put(it) }
        network = NetworkWatch(this) { access -> events.put(CompanionEvent.NetworkChanged(access)) }
        power = PowerWatch(this) { access -> events.put(CompanionEvent.PowerChanged(access)) }
        model = CompanionModel(CompanionState.Stopped, network.access, power.access)
        thread(name = "companion-events") { takeEvents() }
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val event = intent?.let { CompanionIntents.eventOf(it) } ?: CompanionEvent.StartRequested.takeIf { Pairing.paired(this) }
        if (event == null) stopSelf() else events.put(event)
        return START_STICKY
    }

    private fun takeEvents() {
        while (true) {
            val event = events.take()
            if (ended) return
            val step = model.after(event, SystemClock.elapsedRealtime())
            Log.d(LOG_TAG, "$event: ${model.state} -> ${step.model.state}, effects ${step.effects}")
            val changed = step.model.state != model.state
            model = step.model
            if (changed) {
                publish(model.state)
                fitWakeLockTo(model.state)
            }
            step.effects.forEach { carryOut(it) }
        }
    }

    private fun publish(state: CompanionState) {
        CompanionStore.publish(state)
        getSystemService(NotificationManager::class.java).notify(STATUS_NOTIFICATION_ID, statusNotification(this, state))
    }

    @SuppressLint("WakelockTimeout")
    private fun fitWakeLockTo(state: CompanionState) {
        synchronized(wakeLockGuard) {
            when {
                ended -> return
                state.keepsDeviceAwake() && !wakeLock.isHeld -> wakeLock.acquire()
                !state.keepsDeviceAwake() && wakeLock.isHeld -> wakeLock.release()
            }
        }
    }

    private fun carryOut(effect: CompanionEffect) {
        when (effect) {
            is CompanionEffect.BeginPairing -> Pairing.begin(this, effect.link.text)
            CompanionEffect.SendPairingRequest -> pairer.send(Pairing.linkText(this).orEmpty())
            CompanionEffect.CompletePairing -> completePairing()
            CompanionEffect.ForgetPairing -> forgetPairing()
            CompanionEffect.StartServices, CompanionEffect.RestartServices -> services.start(Pairing.linkText(this).orEmpty())
            CompanionEffect.StopServices -> services.stop()
            is CompanionEffect.ScheduleRetry -> retry = after(effect.afterMs, CompanionEvent.RetryDue)
            CompanionEffect.CancelRetry -> retry?.cancel(false)
            is CompanionEffect.SchedulePairingTimeout -> pairingTimeout = after(effect.afterMs, CompanionEvent.PairingTimedOut)
            CompanionEffect.CancelPairingTimeout -> pairingTimeout?.cancel(false)
            is CompanionEffect.PostFailureNotification -> postFailureNotification(this, effect.reason)
            CompanionEffect.CancelFailureNotification -> cancelFailureNotification(this)
            CompanionEffect.EndHost -> end()
        }
    }

    private fun completePairing() {
        Pairing.complete(this)
        pairer.complete()
    }

    private fun forgetPairing() {
        val linkText = Pairing.linkText(this)
        val paired = Pairing.paired(this)
        Pairing.forget(this)
        if (paired && linkText != null) pairer.unpair(linkText) else pairer.giveUp()
    }

    private fun after(delayMs: Long, event: CompanionEvent): ScheduledFuture<*> =
        timers.schedule({ events.put(event) }, delayMs, TimeUnit.MILLISECONDS)

    private fun end() {
        ended = true
        stopSelf()
    }

    private fun keepsShowing(state: CompanionState) = state is CompanionState.PairingFailure || state is CompanionState.Refused

    override fun onDestroy() {
        synchronized(wakeLockGuard) {
            ended = true
            if (wakeLock.isHeld) wakeLock.release()
        }
        events.put(CompanionEvent.StopRequested)
        timers.shutdownNow()
        services.close()
        network.close()
        power.close()
        if (!keepsShowing(CompanionStore.state)) CompanionStore.publish(CompanionState.Stopped)
        super.onDestroy()
    }
}
