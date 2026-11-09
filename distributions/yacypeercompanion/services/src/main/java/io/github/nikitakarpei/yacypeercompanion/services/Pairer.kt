package io.github.nikitakarpei.yacypeercompanion.services

import android.content.Context
import android.os.Build
import android.provider.Settings
import android.util.Log
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent
import io.github.nikitakarpei.yacypeercompanion.model.KeyDisposal
import io.github.nikitakarpei.yacypeercompanion.model.PairingAttempts
import io.github.nikitakarpei.yacypeercompanion.model.PairingProblem
import pairingrequests.Pairingrequests
import kotlin.concurrent.thread

private const val LOG_TAG = "Pairer"

class Pairer(private val ctx: Context, private val events: (CompanionEvent) -> Unit) {
    private val attempts = PairingAttempts()

    fun send(linkText: String) {
        val attempt = attempts.started(linkText)
        thread(name = "pairing") {
            val outcome = pair(linkText)
            when (val disposal = attempts.ended(attempt, outcome)) {
                null -> events(outcome)
                else -> dispose(disposal)
            }
        }
    }

    fun complete() = attempts.completed()

    fun giveUp() {
        val disposal = attempts.givenUp() ?: return
        thread(name = "pairing end") { dispose(disposal) }
    }

    fun unpair(linkText: String) {
        thread(name = "unpairing") { unpairWithKey(linkText) }
    }

    private fun pair(linkText: String): CompanionEvent {
        val challenge = Pairingrequests.challengeFrom(linkText) ?: return CompanionEvent.PairingFailed(PairingProblem.NOT_SENT)
        runCatching { DeviceKeyEntry.generate(challenge) }.onFailure {
            Log.w(LOG_TAG, "device key", it)
            return CompanionEvent.DeviceKeyUnavailable
        }
        val key = DeviceKeyEntry.current() ?: return CompanionEvent.DeviceKeyUnavailable
        val request = Pairingrequests.newRequest(linkText, deviceNameOf(ctx), appVersionOf(ctx))
        DeviceKeyEntry.attestationChain().forEach { request.addCertificate(it) }
        return eventFrom(request.send(key))
    }

    private fun dispose(disposal: KeyDisposal) {
        when (disposal) {
            KeyDisposal.KeepKey -> Log.w(LOG_TAG, "a newer pairing holds the device key, so the app keeps it")

            KeyDisposal.DeleteKey -> DeviceKeyEntry.delete()

            is KeyDisposal.UnpairAndDeleteKey -> {
                Log.w(LOG_TAG, "the peer accepted the device after the app gave up, so the app unpairs it")
                unpairWithKey(disposal.linkText)
            }
        }
    }

    private fun unpairWithKey(linkText: String) {
        val key = DeviceKeyEntry.current()
        if (key == null) {
            Log.w(LOG_TAG, "no device key to send the unpairing request with")
        } else {
            Pairingrequests.unpair(linkText, key)
        }
        DeviceKeyEntry.delete()
    }
}

fun eventFrom(result: String): CompanionEvent = when (result) {
    Pairingrequests.ResultAccepted -> CompanionEvent.PairingAccepted
    Pairingrequests.ResultPeerUnreachable -> CompanionEvent.PairingFailed(PairingProblem.PEER_UNREACHABLE)
    Pairingrequests.ResultSecretExpired -> CompanionEvent.PairingFailed(PairingProblem.SECRET_EXPIRED)
    Pairingrequests.ResultAppOutdated -> CompanionEvent.PairingFailed(PairingProblem.APP_OUTDATED)
    Pairingrequests.ResultPeerOutdated -> CompanionEvent.PairingFailed(PairingProblem.PEER_OUTDATED)
    Pairingrequests.ResultAppNotOfficial -> CompanionEvent.PairingFailed(PairingProblem.APP_NOT_OFFICIAL)
    Pairingrequests.ResultDeviceNotTrusted -> CompanionEvent.PairingFailed(PairingProblem.DEVICE_NOT_TRUSTED)
    Pairingrequests.ResultNotSent -> CompanionEvent.PairingFailed(PairingProblem.NOT_SENT)
    Pairingrequests.ResultReplyUnreadable -> CompanionEvent.PairingFailed(PairingProblem.REPLY_UNREADABLE)
    else -> CompanionEvent.PairingFailed(PairingProblem.REFUSED)
}

fun deviceNameOf(ctx: Context): String = Settings.Global.getString(ctx.contentResolver, Settings.Global.DEVICE_NAME) ?: Build.MODEL

fun appVersionOf(ctx: Context): String = ctx.packageManager.getPackageInfo(ctx.packageName, 0).versionName.orEmpty()
