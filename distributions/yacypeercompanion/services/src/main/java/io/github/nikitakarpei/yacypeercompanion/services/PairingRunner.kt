package io.github.nikitakarpei.yacypeercompanion.services

import android.content.Context
import android.os.Build
import android.os.SystemClock
import android.provider.Settings
import android.util.Log
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent
import io.github.nikitakarpei.yacypeercompanion.model.PAIRING_LIMIT_MS
import io.github.nikitakarpei.yacypeercompanion.model.PairingProblem
import pairing.Outcome
import pairing.Pairing
import kotlin.concurrent.thread

private const val LOG_TAG = "PairingRunner"

class PairingRunner(private val ctx: Context, private val events: (CompanionEvent) -> Unit) {
    fun send(linkText: String) {
        val givenUpAt = SystemClock.elapsedRealtime() + PAIRING_LIMIT_MS
        thread(name = "pairing") { pair(linkText, givenUpAt)?.let(events) }
    }

    fun unpair(linkText: String) {
        thread(name = "unpairing") {
            PairingKey.current()?.let { Pairing.unpair(linkText, it) }
            PairingKey.delete()
        }
    }

    fun discardKey() {
        thread(name = "key discarding") { PairingKey.delete() }
    }

    private fun pair(linkText: String, givenUpAt: Long): CompanionEvent? {
        val challenge = Pairing.challengeOf(linkText) ?: return CompanionEvent.PairingRefused(PairingProblem.REFUSED)
        runCatching { PairingKey.generate(challenge) }.onFailure {
            Log.w(LOG_TAG, "pairing key", it)
            return CompanionEvent.SigningKeyUnavailable
        }
        val key = PairingKey.current() ?: return CompanionEvent.SigningKeyUnavailable
        val request = Pairing.newRequest(linkText, deviceNameOf(ctx), appVersionOf(ctx))
        PairingKey.attestationChain().forEach { request.addCertificate(it) }
        val outcome = request.send(key)
        if (outcome.result == Pairing.ResultAccepted && SystemClock.elapsedRealtime() >= givenUpAt) {
            Log.w(LOG_TAG, "the peer accepted the device after the app gave up, so the app unpairs it")
            Pairing.unpair(linkText, key)
            return null
        }
        return eventFrom(outcome)
    }
}

fun eventFrom(outcome: Outcome): CompanionEvent = when (outcome.result) {
    Pairing.ResultAccepted -> CompanionEvent.PairingAccepted
    Pairing.ResultNoAnswer -> CompanionEvent.PairingRefused(PairingProblem.PEER_UNREACHABLE)
    Pairing.ResultSecretExpired -> CompanionEvent.PairingRefused(PairingProblem.SECRET_EXPIRED)
    Pairing.ResultAppOutdated -> CompanionEvent.PairingRefused(PairingProblem.APP_OUTDATED)
    Pairing.ResultPeerOutdated -> CompanionEvent.PairingRefused(PairingProblem.PEER_OUTDATED)
    Pairing.ResultAppNotOfficial -> CompanionEvent.PairingRefused(PairingProblem.APP_NOT_OFFICIAL)
    Pairing.ResultDeviceNotTrusted -> CompanionEvent.PairingRefused(PairingProblem.DEVICE_NOT_TRUSTED)
    else -> CompanionEvent.PairingRefused(PairingProblem.REFUSED)
}

fun deviceNameOf(ctx: Context): String = Settings.Global.getString(ctx.contentResolver, Settings.Global.DEVICE_NAME) ?: Build.MODEL

fun appVersionOf(ctx: Context): String = ctx.packageManager.getPackageInfo(ctx.packageName, 0).versionName.orEmpty()
