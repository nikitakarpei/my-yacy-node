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
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.SendPairingRequest
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.StartServices
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEffect.StopServices
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.BrokerConnectionLost
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.CancelPairing
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.NetworkChanged
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.PairWith
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.PairingAccepted
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.PairingRefused
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.PairingTimedOut
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.PowerChanged
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.RetryDue
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.ServicesConnected
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.ServicesEnded
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.SigningKeyUnavailable
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.StartRequested
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.StopRequested
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.Unpair
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Connected
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Connecting
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.DeviceRefused
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Pairing
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.PairingFailed
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Reconnecting
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.RetryingAt
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Stopped
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.WaitingForCharger
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.WaitingForNetwork
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.WaitingForUnmeteredNetwork
import org.junit.Assert.assertEquals
import org.junit.Test

class CompanionMachineTest {
    private val now = 1_000_000L
    private val code = PairingCode("yacypeer-companion://pair?address=192.168.1.20%3A4223&brokerKeyHash=hash&secret=secret")
    private val unreachable = ServicesEnded(ServicesEnding.BROKER_UNREACHABLE)
    private val lostPeer = DisconnectReason.PEER_UNREACHABLE

    private val usable = NetworkChanged(NetworkAccess.USABLE)
    private val meteredOnly = NetworkChanged(NetworkAccess.METERED_ONLY)
    private val onCharger = PowerChanged(PowerAccess.USABLE)
    private val offCharger = PowerChanged(PowerAccess.CHARGER_NEEDED)

    private fun stepOf(state: CompanionState, event: CompanionEvent, networkAccess: NetworkAccess = NetworkAccess.USABLE) =
        CompanionModel(state, networkAccess).after(event, now)

    private fun offChargerStepOf(state: CompanionState, event: CompanionEvent) =
        CompanionModel(state, powerAccess = PowerAccess.CHARGER_NEEDED).after(event, now)

    private fun assertStep(step: Step, state: CompanionState, vararg effects: CompanionEffect) {
        assertEquals(state, step.model.state)
        assertEquals(effects.toList(), step.effects)
    }

    @Test
    fun pairingSendsThePairingRequestWithATimeLimit() {
        assertStep(
            stepOf(Stopped, PairWith(code)),
            Pairing,
            BeginPairing(code),
            SendPairingRequest,
            SchedulePairingTimeout(PAIRING_LIMIT_MS),
        )
    }

    @Test
    fun anAcceptedPairingStartsTheServices() {
        assertStep(stepOf(Pairing, PairingAccepted), Connecting, CancelPairingTimeout, CompletePairing, StartServices)
    }

    @Test
    fun pairingFailsWithoutRetriesAndEndsTheHost() {
        assertStep(
            stepOf(Pairing, PairingRefused(PairingProblem.APP_OUTDATED)),
            PairingFailed(PairingProblem.APP_OUTDATED),
            CancelPairingTimeout,
            EndHost,
        )
        assertStep(
            stepOf(Pairing, SigningKeyUnavailable),
            PairingFailed(PairingProblem.SIGNING_KEY),
            CancelPairingTimeout,
            EndHost,
        )
        assertStep(stepOf(Pairing, PairingTimedOut), PairingFailed(PairingProblem.PEER_UNREACHABLE), EndHost)
    }

    @Test
    fun anOutcomeOutsideAPairingChangesNothing() {
        assertStep(stepOf(Connected, PairingAccepted), Connected)
        assertStep(stepOf(Stopped, PairingRefused(PairingProblem.REFUSED)), Stopped)
    }

    @Test
    fun aDeviceThatThePeerUnpairedStaysDisconnectedAndTellsWhy() {
        assertStep(
            stepOf(Connected, ServicesEnded(ServicesEnding.DEVICE_REFUSED)),
            DeviceRefused,
            StopServices,
            CancelRetry,
            PostFailureNotification(DisconnectReason.DEVICE_REFUSED),
            EndHost,
        )
        assertStep(stepOf(DeviceRefused, StartRequested), DeviceRefused)
        assertStep(stepOf(Stopped, ServicesEnded(ServicesEnding.DEVICE_REFUSED)), Stopped)
    }

    @Test
    fun pairingAgainAfterAFailureStartsOver() {
        assertEquals(Pairing, stepOf(PairingFailed(PairingProblem.PEER_UNREACHABLE), PairWith(code)).model.state)
    }

    @Test
    fun cancellingPairingForgetsTheCode() {
        assertStep(stepOf(Pairing, CancelPairing), Stopped, StopServices, CancelPairingTimeout, ForgetPairing, EndHost)
    }

    @Test
    fun aLostConnectionPostsOneFailureNotification() {
        assertStep(stepOf(Connected, BrokerConnectionLost), Reconnecting(lostPeer, 0), PostFailureNotification(lostPeer))
        val gaveUp = stepOf(Reconnecting(lostPeer, 0), unreachable)
        assertStep(gaveUp, RetryingAt(lostPeer, now + FIRST_RETRY_MS, 1), ScheduleRetry(FIRST_RETRY_MS))
    }

    @Test
    fun aFailedStartAfterConnectingPostsTheFailureNotification() {
        assertStep(
            stepOf(Connecting, unreachable),
            RetryingAt(lostPeer, now + FIRST_RETRY_MS, 1),
            PostFailureNotification(lostPeer),
            ScheduleRetry(FIRST_RETRY_MS),
        )
    }

    @Test
    fun theRetryPauseDoublesUpToAMinute() {
        val pauses = (0..6).map { retryPauseAfter(it) }
        assertEquals(listOf(5_000L, 10_000L, 20_000L, 40_000L, 60_000L, 60_000L, 60_000L), pauses)
        assertStep(stepOf(Reconnecting(lostPeer, 3), unreachable), RetryingAt(lostPeer, now + 40_000L, 4), ScheduleRetry(40_000L))
    }

    @Test
    fun aDueRetryStartsTheServices() {
        assertStep(stepOf(RetryingAt(lostPeer, now, 2), RetryDue), Reconnecting(lostPeer, 2), StartServices)
    }

    @Test
    fun connectingAgainClearsTheFailure() {
        assertStep(stepOf(Reconnecting(lostPeer, 4), ServicesConnected), Connected, CancelRetry, CancelFailureNotification)
    }

    @Test
    fun withoutANetworkTheCompanionWaitsForOne() {
        assertStep(stepOf(Reconnecting(lostPeer, 1), unreachable, NetworkAccess.NONE), WaitingForNetwork(lostPeer, 2))
        assertStep(stepOf(RetryingAt(lostPeer, now, 2), NetworkChanged(NetworkAccess.NONE)), WaitingForNetwork(lostPeer, 2), CancelRetry)
        assertStep(stepOf(WaitingForNetwork(lostPeer, 2), usable, NetworkAccess.NONE), Reconnecting(lostPeer, 0), StartServices)
    }

    @Test
    fun withOnlyMeteredNetworksTheCompanionStopsAndWaitsForAnUnmeteredOne() {
        assertStep(stepOf(Connected, meteredOnly), WaitingForUnmeteredNetwork, StopServices, CancelRetry, CancelFailureNotification)
        assertStep(
            stepOf(RetryingAt(lostPeer, now, 2), meteredOnly),
            WaitingForUnmeteredNetwork,
            StopServices,
            CancelRetry,
            CancelFailureNotification,
        )
        assertStep(stepOf(Stopped, StartRequested, NetworkAccess.METERED_ONLY), WaitingForUnmeteredNetwork)
        assertStep(stepOf(Reconnecting(lostPeer, 1), unreachable, NetworkAccess.METERED_ONLY), WaitingForUnmeteredNetwork)
        assertStep(stepOf(WaitingForUnmeteredNetwork, usable, NetworkAccess.METERED_ONLY), Connecting, StartServices)
    }

    @Test
    fun pairingOverAMeteredNetworkCompletesAndThenWaits() {
        assertStep(
            stepOf(Pairing, PairingAccepted, NetworkAccess.METERED_ONLY),
            WaitingForUnmeteredNetwork,
            CancelPairingTimeout,
            CompletePairing,
        )
    }

    @Test
    fun aNetworkChangeTriesAtOnceAndStartsThePauseAgain() {
        assertStep(stepOf(RetryingAt(lostPeer, now + 40_000L, 4), usable), Reconnecting(lostPeer, 0), CancelRetry, StartServices)
        assertStep(stepOf(Reconnecting(lostPeer, 4), usable), Reconnecting(lostPeer, 0), RestartServices)
        assertStep(stepOf(Connected, usable), Connected, RestartServices)
    }

    @Test
    fun aStopThatWasAskedForChangesNothing() {
        assertStep(stepOf(Connected, ServicesEnded(ServicesEnding.STOPPED)), Connected)
    }

    @Test
    fun theUserStopsOrUnpairsFromAnyState() {
        assertStep(stepOf(RetryingAt(lostPeer, now, 1), StopRequested), Stopped, StopServices, CancelRetry, CancelPairingTimeout, EndHost)
        assertStep(
            stepOf(Connected, Unpair),
            Stopped,
            StopServices,
            CancelRetry,
            CancelPairingTimeout,
            CancelFailureNotification,
            ForgetPairing,
            EndHost,
        )
    }

    @Test
    fun openingTheAppStartsAStoppedCompanion() {
        assertStep(stepOf(Stopped, StartRequested), Connecting, StartServices)
        assertStep(stepOf(Connected, StartRequested), Connected)
    }

    @Test
    fun withoutAChargerTheCompanionStopsAndWaitsForOne() {
        assertStep(stepOf(Connected, offCharger), WaitingForCharger, StopServices, CancelRetry, CancelFailureNotification)
        assertStep(
            stepOf(RetryingAt(lostPeer, now, 2), offCharger),
            WaitingForCharger,
            StopServices,
            CancelRetry,
            CancelFailureNotification,
        )
        assertStep(offChargerStepOf(Stopped, StartRequested), WaitingForCharger)
        assertStep(
            stepOf(WaitingForUnmeteredNetwork, offCharger),
            WaitingForCharger,
            StopServices,
            CancelRetry,
            CancelFailureNotification,
        )
    }

    @Test
    fun aChargerStartsAWaitingCompanion() {
        assertStep(offChargerStepOf(WaitingForCharger, onCharger), Connecting, StartServices)
        assertStep(stepOf(WaitingForCharger, onCharger, NetworkAccess.METERED_ONLY), WaitingForUnmeteredNetwork)
    }

    @Test
    fun aNetworkChangeLeavesACompanionWaitingForACharger() {
        assertStep(offChargerStepOf(WaitingForCharger, usable), WaitingForCharger)
    }

    @Test
    fun pairingWithoutAChargerCompletesAndThenWaits() {
        assertStep(offChargerStepOf(Pairing, PairingAccepted), WaitingForCharger, CancelPairingTimeout, CompletePairing)
    }
}
