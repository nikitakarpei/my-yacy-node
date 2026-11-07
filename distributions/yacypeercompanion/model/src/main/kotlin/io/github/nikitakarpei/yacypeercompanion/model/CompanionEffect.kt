package io.github.nikitakarpei.yacypeercompanion.model

sealed interface CompanionEffect {
    data class BeginPairing(val code: PairingCode) : CompanionEffect

    data object SendPairingRequest : CompanionEffect

    data object CompletePairing : CompanionEffect

    data object ForgetPairing : CompanionEffect

    data object StartServices : CompanionEffect

    data object StopServices : CompanionEffect

    data object RestartServices : CompanionEffect

    data class ScheduleRetry(val afterMs: Long) : CompanionEffect

    data object CancelRetry : CompanionEffect

    data class SchedulePairingTimeout(val afterMs: Long) : CompanionEffect

    data object CancelPairingTimeout : CompanionEffect

    data class PostFailureNotification(val reason: DisconnectReason) : CompanionEffect

    data object CancelFailureNotification : CompanionEffect

    data object EndHost : CompanionEffect
}
