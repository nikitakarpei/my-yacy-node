package io.github.nikitakarpei.yacypeercompanion.model

sealed interface CompanionEvent {
    data class PairWith(val code: PairingCode) : CompanionEvent

    data object CancelPairing : CompanionEvent

    data object StartRequested : CompanionEvent

    data object StopRequested : CompanionEvent

    data object Unpair : CompanionEvent

    data object ServicesConnected : CompanionEvent

    data object BrokerConnectionLost : CompanionEvent

    data class ServicesEnded(val ending: ServicesEnding) : CompanionEvent

    data object SigningKeyUnavailable : CompanionEvent

    data class NetworkChanged(val access: NetworkAccess) : CompanionEvent

    data object RetryDue : CompanionEvent

    data object PairingTimedOut : CompanionEvent
}

enum class ServicesEnding { STOPPED, BROKER_UNREACHABLE, FAILED }
