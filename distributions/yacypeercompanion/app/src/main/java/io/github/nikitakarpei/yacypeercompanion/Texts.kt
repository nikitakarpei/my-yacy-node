package io.github.nikitakarpei.yacypeercompanion

import android.os.SystemClock
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState
import io.github.nikitakarpei.yacypeercompanion.model.DisconnectReason
import io.github.nikitakarpei.yacypeercompanion.model.PairingProblem

fun titleOf(state: CompanionState): String = when (state) {
    CompanionState.Stopped -> "Stopped"
    CompanionState.Pairing, is CompanionState.PairingFailure -> "Pairing"
    CompanionState.Connecting -> "Connecting"
    CompanionState.Connected -> "Connected"
    CompanionState.WaitingForCharger -> "Waiting for a charger"
    CompanionState.WaitingForUnmeteredNetwork -> "Waiting for Wi-Fi"
    is CompanionState.Disconnected -> "Disconnected"
}

fun textOf(reason: DisconnectReason): String = when (reason) {
    DisconnectReason.PEER_UNREACHABLE -> "Lost connection to the peer"
    DisconnectReason.SIGNING_KEY -> "Couldn’t create a signing key on this device"
    DisconnectReason.DEVICE_KEY -> "Couldn’t find the key that paired this device"
    DisconnectReason.SERVICES_FAILED -> "Something went wrong"
    DisconnectReason.DEVICE_REFUSED -> "The peer no longer accepts this device"
}

fun reasonTextOf(state: CompanionState.Disconnected): String = when (state) {
    CompanionState.DeviceRefused -> textOf(DisconnectReason.DEVICE_REFUSED)
    is CompanionState.Offline -> "No network connection"
    is CompanionState.Recovering -> textOf(state.reason)
}

const val RECONNECTING_TEXT = "Reconnecting"

const val DATA_SHARING_TITLE = "Data sharing"

fun attemptTextOf(state: CompanionState.Disconnected): String? = when (state) {
    is CompanionState.Reconnecting -> RECONNECTING_TEXT
    is CompanionState.RetryingAt -> "Retrying in ${((state.retryAtMs - SystemClock.elapsedRealtime()) / 1_000).coerceAtLeast(0)} s"
    is CompanionState.Offline, CompanionState.DeviceRefused -> null
}

fun textOf(problem: PairingProblem): String = when (problem) {
    PairingProblem.PEER_UNREACHABLE -> "The peer doesn’t answer. Make sure this device and the peer are on the same network."
    PairingProblem.DEVICE_KEY -> "Couldn’t create a key for pairing on this device."
    PairingProblem.SECRET_EXPIRED -> "The QR code has expired. Scan a new one."
    PairingProblem.APP_NOT_OFFICIAL -> "The peer accepts only official builds of the app."
    PairingProblem.APP_OUTDATED -> "This app is outdated. Please update the app."
    PairingProblem.PEER_OUTDATED -> "This app requires a newer YaCy peer. Please update the YaCy peer."
    PairingProblem.DEVICE_NOT_TRUSTED -> "The peer doesn’t accept this device."
    PairingProblem.REFUSED -> "The peer refused the pairing."
    PairingProblem.NOT_SENT -> "The app couldn’t send the pairing request."
    PairingProblem.REPLY_UNREADABLE -> "The app couldn’t read the reply of the peer."
}

fun retriesWithSameLink(problem: PairingProblem): Boolean = problem == PairingProblem.PEER_UNREACHABLE
