package io.github.nikitakarpei.yacypeercompanion.model

sealed interface CompanionState {
    data object Stopped : CompanionState

    data object Pairing : CompanionState

    data class PairingFailure(val problem: PairingProblem) : CompanionState

    data object Connecting : CompanionState

    data object Connected : CompanionState

    sealed interface Waiting : CompanionState

    data object WaitingForCharger : Waiting

    data object WaitingForUnmeteredNetwork : Waiting

    sealed interface Disconnected : CompanionState

    data class Refused(val reason: DisconnectReason) : Disconnected

    sealed interface Recovering : Disconnected {
        val reason: DisconnectReason
        val failedAttempts: Int
    }

    data class Reconnecting(override val reason: DisconnectReason, override val failedAttempts: Int) : Recovering

    data class RetryingAt(override val reason: DisconnectReason, val retryAtMs: Long, override val failedAttempts: Int) : Recovering

    data class Offline(override val reason: DisconnectReason, override val failedAttempts: Int) : Recovering
}

enum class NetworkAccess { USABLE, NONE, METERED_ONLY }

enum class PowerAccess { USABLE, BATTERY_ONLY }

enum class DisconnectReason { PEER_UNREACHABLE, SIGNING_KEY, DEVICE_KEY, SERVICES_FAILED, DEVICE_REFUSED, APP_OUTDATED, PEER_OUTDATED }

enum class PairingProblem {
    PEER_UNREACHABLE,
    DEVICE_KEY,
    SECRET_EXPIRED,
    APP_NOT_OFFICIAL,
    APP_OUTDATED,
    PEER_OUTDATED,
    DEVICE_NOT_TRUSTED,
    REFUSED,
    NOT_SENT,
    REPLY_UNREADABLE,
}

class PairLink(val text: String)

fun CompanionState.keepsDeviceAwake(): Boolean = this == CompanionState.Pairing || this == CompanionState.Connecting ||
    this == CompanionState.Connected || this is CompanionState.Recovering
