package io.github.nikitakarpei.yacypeercompanion

import android.os.SystemClock
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState
import io.github.nikitakarpei.yacypeercompanion.model.DisconnectReason
import io.github.nikitakarpei.yacypeercompanion.model.PairingProblem

fun titleOf(state: CompanionState): String = when (state) {
    CompanionState.Stopped -> "Stopped"
    CompanionState.Pairing, is CompanionState.PairingFailed -> "Pairing"
    CompanionState.Connecting -> "Connecting"
    CompanionState.Connected -> "Connected"
    is CompanionState.Disconnected -> "Disconnected"
}

fun textOf(reason: DisconnectReason): String = when (reason) {
    DisconnectReason.PEER_UNREACHABLE -> "Lost connection to the peer"
    DisconnectReason.SIGNING_KEY -> "Couldn’t create a signing key on this phone"
    DisconnectReason.SERVICES_FAILED -> "Something went wrong"
}

fun reasonTextOf(state: CompanionState.Disconnected): String = when (state) {
    CompanionState.WaitingForUnmeteredNetwork -> "No Wi-Fi connection"
    is CompanionState.WaitingForNetwork -> "No network connection"
    is CompanionState.Recovering -> textOf(state.reason)
}

const val RECONNECTING_TEXT = "Reconnecting"

fun attemptTextOf(state: CompanionState.Disconnected): String? = when (state) {
    is CompanionState.Reconnecting -> RECONNECTING_TEXT
    is CompanionState.RetryingAt -> "Retrying in ${((state.retryAtMs - SystemClock.elapsedRealtime()) / 1_000).coerceAtLeast(0)} s"
    is CompanionState.WaitingForNetwork, CompanionState.WaitingForUnmeteredNetwork -> null
}

fun textOf(problem: PairingProblem): String = when (problem) {
    PairingProblem.PEER_UNREACHABLE -> "Couldn’t connect to the peer. Make sure this phone and the peer are on the same network."

    PairingProblem.SIGNING_KEY ->
        "Couldn’t create a signing key on this phone. This phone can’t sign the pages it fetches, so it can’t pair."

    PairingProblem.SERVICES_FAILED -> "Couldn’t pair with the peer."
}

fun retriesWithSameCode(problem: PairingProblem): Boolean = problem == PairingProblem.PEER_UNREACHABLE
