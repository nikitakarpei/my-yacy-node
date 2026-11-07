package io.github.nikitakarpei.yacypeercompanion.services

import android.content.Context
import android.os.Build
import android.provider.Settings
import android.util.Log
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent
import io.github.nikitakarpei.yacypeercompanion.model.PairingProblem
import pairing.Outcome
import pairing.Pairing
import kotlin.concurrent.thread

private const val LOG_TAG = "PairingRunner"

class PairingRunner(private val ctx: Context, private val events: (CompanionEvent) -> Unit) {
    fun send(linkText: String) {
        thread(name = "pairing") { events(eventFor(linkText)) }
    }

    fun forget(linkText: String?, paired: Boolean) {
        thread(name = "unpairing") {
            val key = PairingKey.current()
            if (paired && linkText != null && key != null) Pairing.unpair(linkText, key)
            PairingKey.delete()
        }
    }

    private fun eventFor(linkText: String): CompanionEvent {
        val challenge = Pairing.challengeOf(linkText) ?: return CompanionEvent.PairingRefused(PairingProblem.REFUSED)
        val attestation =
            runCatching { PairingKey.madeFor(challenge) }.getOrElse {
                Log.w(LOG_TAG, "pairing key", it)
                return CompanionEvent.SigningKeyUnavailable
            }
        val key = PairingKey.current() ?: return CompanionEvent.SigningKeyUnavailable
        val request = Pairing.newRequest(linkText, deviceNameOf(ctx), appVersionOf(ctx))
        attestation.forEach { request.addCertificate(it) }
        return eventOf(request.send(key))
    }
}

fun eventOf(outcome: Outcome): CompanionEvent = when (outcome.result) {
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
