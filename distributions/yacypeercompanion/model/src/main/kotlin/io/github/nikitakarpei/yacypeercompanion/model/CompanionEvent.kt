package io.github.nikitakarpei.yacypeercompanion.model

sealed interface CompanionEvent {
    data class PairWith(val link: PairLink) : CompanionEvent

    data object CancelPairing : CompanionEvent

    data object PairingAccepted : CompanionEvent

    data class PairingFailed(val problem: PairingProblem) : CompanionEvent

    data object StartRequested : CompanionEvent

    data object StopRequested : CompanionEvent

    data object Unpair : CompanionEvent

    data object ServicesConnected : CompanionEvent

    data object BrokerConnectionLost : CompanionEvent

    data class ServicesEnded(val ending: ServicesEnding) : CompanionEvent

    data object SigningKeyUnavailable : CompanionEvent

    data object DeviceKeyUnavailable : CompanionEvent

    data class NetworkChanged(val access: NetworkAccess) : CompanionEvent

    data class PowerChanged(val access: PowerAccess) : CompanionEvent

    data object RetryDue : CompanionEvent

    data object PairingTimedOut : CompanionEvent
}

enum class ServicesEnding { STOPPED, BROKER_UNREACHABLE, FAILED, DEVICE_REFUSED }
