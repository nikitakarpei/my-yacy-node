package io.github.nikitakarpei.yacypeercompanion.services

import android.content.Context
import android.os.Build
import android.provider.Settings
import android.util.Log
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent
import io.github.nikitakarpei.yacypeercompanion.model.PairingProblem
import pairing.Pairing
import kotlin.concurrent.thread

private const val LOG_TAG = "Pairer"

class Pairer(private val ctx: Context, private val events: (CompanionEvent) -> Unit) {
    private val lock = Any()
    private var attempt: Attempt? = null

    fun send(linkText: String) {
        val started = Attempt(linkText)
        synchronized(lock) { attempt = started }
        thread(name = "pairing") { finish(started, pair(linkText)) }
    }

    fun complete() {
        synchronized(lock) { attempt = null }
    }

    fun giveUp() {
        var running = false
        val ended = synchronized(lock) {
            val current = attempt
            attempt = null
            current?.givenUp = true
            running = current != null && current.event == null
            current?.takeIf { it.event != null }
        }
        when {
            ended != null -> thread(name = "pairing end") { dispose(ended) }
            !running -> thread(name = "key discarding") { DeviceKeyEntry.delete() }
        }
    }

    fun unpair(linkText: String) {
        thread(name = "unpairing") { unpairWithKey(linkText) }
    }

    private fun pair(linkText: String): CompanionEvent {
        val challenge = Pairing.challengeFrom(linkText) ?: return CompanionEvent.PairingFailed(PairingProblem.NOT_SENT)
        runCatching { DeviceKeyEntry.generate(challenge) }.onFailure {
            Log.w(LOG_TAG, "device key", it)
            return CompanionEvent.DeviceKeyUnavailable
        }
        val key = DeviceKeyEntry.current() ?: return CompanionEvent.DeviceKeyUnavailable
        val request = Pairing.newRequest(linkText, deviceNameOf(ctx), appVersionOf(ctx))
        DeviceKeyEntry.attestationChain().forEach { request.addCertificate(it) }
        return eventFrom(request.send(key))
    }

    private fun finish(started: Attempt, event: CompanionEvent) {
        val givenUp = synchronized(lock) {
            started.event = event
            if (!started.givenUp) events(event)
            started.givenUp
        }
        if (givenUp) dispose(started)
    }

    private fun dispose(ended: Attempt) {
        if (synchronized(lock) { attempt != null }) {
            Log.w(LOG_TAG, "a newer pairing holds the device key, so the app keeps it")
            return
        }
        if (ended.event == CompanionEvent.PairingAccepted) {
            Log.w(LOG_TAG, "the peer accepted the device after the app gave up, so the app unpairs it")
            unpairWithKey(ended.linkText)
            return
        }
        DeviceKeyEntry.delete()
    }

    private fun unpairWithKey(linkText: String) {
        val key = DeviceKeyEntry.current()
        if (key == null) {
            Log.w(LOG_TAG, "no device key to send the unpairing request with")
        } else {
            Pairing.unpair(linkText, key)
        }
        DeviceKeyEntry.delete()
    }

    private class Attempt(val linkText: String) {
        var givenUp = false
        var event: CompanionEvent? = null
    }
}

fun eventFrom(result: String): CompanionEvent = when (result) {
    Pairing.ResultAccepted -> CompanionEvent.PairingAccepted
    Pairing.ResultPeerUnreachable -> CompanionEvent.PairingFailed(PairingProblem.PEER_UNREACHABLE)
    Pairing.ResultSecretExpired -> CompanionEvent.PairingFailed(PairingProblem.SECRET_EXPIRED)
    Pairing.ResultAppOutdated -> CompanionEvent.PairingFailed(PairingProblem.APP_OUTDATED)
    Pairing.ResultPeerOutdated -> CompanionEvent.PairingFailed(PairingProblem.PEER_OUTDATED)
    Pairing.ResultAppNotOfficial -> CompanionEvent.PairingFailed(PairingProblem.APP_NOT_OFFICIAL)
    Pairing.ResultDeviceNotTrusted -> CompanionEvent.PairingFailed(PairingProblem.DEVICE_NOT_TRUSTED)
    Pairing.ResultNotSent -> CompanionEvent.PairingFailed(PairingProblem.NOT_SENT)
    Pairing.ResultReplyUnreadable -> CompanionEvent.PairingFailed(PairingProblem.REPLY_UNREADABLE)
    else -> CompanionEvent.PairingFailed(PairingProblem.REFUSED)
}

fun deviceNameOf(ctx: Context): String = Settings.Global.getString(ctx.contentResolver, Settings.Global.DEVICE_NAME) ?: Build.MODEL

fun appVersionOf(ctx: Context): String = ctx.packageManager.getPackageInfo(ctx.packageName, 0).versionName.orEmpty()
