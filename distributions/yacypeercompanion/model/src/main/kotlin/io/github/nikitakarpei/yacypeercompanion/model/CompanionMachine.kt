package io.github.nikitakarpei.yacypeercompanion.model

import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.BeginPairing
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.CancelFailureNotification
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.CancelPairingTimeout
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.CancelRetry
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.CompletePairing
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.EndHost
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.ForgetPairing
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.PostFailureNotification
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.RestartServices
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.SchedulePairingTimeout
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.ScheduleRetry
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.StartServices
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.StopServices
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.BrokerConnectionLost
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.CancelPairing
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.NetworkChanged
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.PairWith
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.PairingTimedOut
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.RetryDue
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.ServicesConnected
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.ServicesEnded
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.SigningKeyUnavailable
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.StartRequested
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.StopRequested
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.Unpair
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Connected
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Connecting
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Disconnected
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Pairing
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.PairingFailed
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Reconnecting
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Recovering
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.RetryingAt
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Stopped
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.WaitingForNetwork
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.WaitingForUnmeteredNetwork

const val PAIRING_LIMIT_MS = 30_000L
const val FIRST_RETRY_MS = 5_000L
const val LONGEST_RETRY_MS = 60_000L
private const val RETRY_DOUBLINGS = 4

data class CompanionModel(val state: CompanionState = Stopped, val networkAccess: NetworkAccess = NetworkAccess.USABLE)

data class Step(val model: CompanionModel, val effects: List<CompanionEffect>)

fun CompanionModel.after(event: CompanionEvent, nowMs: Long): Step {
    val model = if (event is NetworkChanged) copy(networkAccess = event.access) else this
    return model.stateAfter(event, nowMs) ?: Step(model, emptyList())
}

fun retryPauseAfter(failedAttempts: Int): Long =
    (FIRST_RETRY_MS shl failedAttempts.coerceAtMost(RETRY_DOUBLINGS)).coerceAtMost(LONGEST_RETRY_MS)

private fun CompanionModel.stateAfter(event: CompanionEvent, nowMs: Long): Step? = when (event) {
    is PairWith -> pairingWith(event.code)
    CancelPairing -> pairingCancelled()
    StartRequested -> started().takeIf { state == Stopped }
    StopRequested -> to(Stopped, StopServices, CancelRetry, CancelPairingTimeout, EndHost)
    Unpair -> to(Stopped, StopServices, CancelRetry, CancelPairingTimeout, CancelFailureNotification, ForgetPairing, EndHost)
    ServicesConnected -> servicesConnected()
    BrokerConnectionLost -> connectionLost()
    is ServicesEnded -> servicesEnded(event.ending, nowMs)
    SigningKeyUnavailable -> failed(DisconnectReason.SIGNING_KEY, PairingProblem.SIGNING_KEY, nowMs)
    is NetworkChanged -> networkChanged()
    RetryDue -> (state as? RetryingAt)?.let { to(Reconnecting(it.reason, it.failedAttempts), StartServices) }
    PairingTimedOut -> to(PairingFailed(PairingProblem.SERVICES_FAILED), StopServices, EndHost).takeIf { state == Pairing }
}

private fun CompanionModel.pairingWith(code: PairingCode): Step? =
    to(Pairing, BeginPairing(code), StartServices, SchedulePairingTimeout(PAIRING_LIMIT_MS))
        .takeIf { state == Stopped || state is PairingFailed }

private fun CompanionModel.pairingCancelled(): Step? = to(Stopped, StopServices, CancelPairingTimeout, ForgetPairing, EndHost)
    .takeIf { state == Stopped || state == Pairing || state is PairingFailed }

private fun CompanionModel.started(): Step =
    if (networkAccess == NetworkAccess.METERED_ONLY) to(WaitingForUnmeteredNetwork) else to(Connecting, StartServices)

private fun CompanionModel.servicesConnected(): Step? = when (state) {
    Pairing -> paired()
    Connecting -> to(Connected)
    is Recovering -> to(Connected, CancelRetry, CancelFailureNotification)
    else -> null
}

private fun CompanionModel.paired(): Step {
    if (networkAccess != NetworkAccess.METERED_ONLY) return to(Connected, CancelPairingTimeout, CompletePairing)
    return to(WaitingForUnmeteredNetwork, CancelPairingTimeout, CompletePairing, StopServices)
}

private fun CompanionModel.connectionLost(): Step? = when (state) {
    Connecting, Connected -> to(
        Reconnecting(DisconnectReason.PEER_UNREACHABLE, 0),
        PostFailureNotification(DisconnectReason.PEER_UNREACHABLE),
    )

    else -> null
}

private fun CompanionModel.servicesEnded(ending: ServicesEnding, nowMs: Long): Step? = when (ending) {
    ServicesEnding.STOPPED -> null
    ServicesEnding.BROKER_UNREACHABLE -> failed(DisconnectReason.PEER_UNREACHABLE, PairingProblem.PEER_UNREACHABLE, nowMs)
    ServicesEnding.FAILED -> failed(DisconnectReason.SERVICES_FAILED, PairingProblem.SERVICES_FAILED, nowMs)
}

private fun CompanionModel.failed(reason: DisconnectReason, pairingProblem: PairingProblem, nowMs: Long): Step? {
    val current = state
    if (current == Pairing) return to(PairingFailed(pairingProblem), StopServices, CancelPairingTimeout, EndHost)
    if (current != Connecting && current != Connected && current !is Recovering) return null
    val failedAttempts = (current as? Recovering)?.failedAttempts ?: 0
    val notice = if (current is Recovering) emptyList() else listOf(PostFailureNotification(reason))
    if (networkAccess == NetworkAccess.METERED_ONLY) return to(WaitingForUnmeteredNetwork, notice)
    if (networkAccess == NetworkAccess.NONE) return to(WaitingForNetwork(reason, failedAttempts + 1), notice)
    val pause = retryPauseAfter(failedAttempts)
    return to(RetryingAt(reason, nowMs + pause, failedAttempts + 1), notice + ScheduleRetry(pause))
}

private fun CompanionModel.networkChanged(): Step? = when (networkAccess) {
    NetworkAccess.USABLE -> usableNetworkChanged()
    NetworkAccess.NONE -> networkLost()
    NetworkAccess.METERED_ONLY -> meteredOnly()
}

private fun CompanionModel.usableNetworkChanged(): Step? = when (val current = state) {
    Connected -> to(Connected, RestartServices)
    WaitingForUnmeteredNetwork -> to(Connecting, StartServices)
    is Reconnecting -> to(Reconnecting(current.reason, 0), RestartServices)
    is RetryingAt -> to(Reconnecting(current.reason, 0), CancelRetry, StartServices)
    is WaitingForNetwork -> to(Reconnecting(current.reason, 0), StartServices)
    else -> null
}

private fun CompanionModel.networkLost(): Step? = when (val current = state) {
    is RetryingAt -> to(WaitingForNetwork(current.reason, current.failedAttempts), CancelRetry)
    else -> null
}

private fun CompanionModel.meteredOnly(): Step? = when (state) {
    Connecting, Connected, is Recovering -> to(WaitingForUnmeteredNetwork, StopServices, CancelRetry, CancelFailureNotification)
    else -> null
}

private fun CompanionModel.to(next: CompanionState, vararg effects: CompanionEffect) = Step(copy(state = next), effects.toList())

private fun CompanionModel.to(next: CompanionState, effects: List<CompanionEffect>) = Step(copy(state = next), effects)
