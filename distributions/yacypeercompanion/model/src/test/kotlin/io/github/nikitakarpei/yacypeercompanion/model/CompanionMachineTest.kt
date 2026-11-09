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
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.DeviceKeyUnavailable
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.NetworkChanged
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.PairWith
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.PairingAccepted
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.PairingFailed
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
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Offline
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Pairing
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.PairingFailure
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Reconnecting
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Refused
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.RetryingAt
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.Stopped
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.WaitingForCharger
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState.WaitingForUnmeteredNetwork
import org.junit.Assert.assertEquals
import org.junit.Test

class CompanionMachineTest {
    private val now = 1_000_000L
    private val link = PairLink("yacypeer-companion://pair?address=192.168.1.20%3A4223&brokerKeyHash=hash&secret=secret")
    private val unreachable = ServicesEnded(ServicesEnding.BROKER_UNREACHABLE)
    private val lostPeer = DisconnectReason.PEER_UNREACHABLE

    private val usableNetwork = NetworkChanged(NetworkAccess.USABLE)
    private val meteredNetworkOnly = NetworkChanged(NetworkAccess.METERED_ONLY)
    private val usablePower = PowerChanged(PowerAccess.USABLE)
    private val batteryOnly = PowerChanged(PowerAccess.BATTERY_ONLY)

    private fun stepOf(
        state: CompanionState,
        event: CompanionEvent,
        networkAccess: NetworkAccess = NetworkAccess.USABLE,
        powerAccess: PowerAccess = PowerAccess.USABLE,
    ) = CompanionModel(state, networkAccess, powerAccess).after(event, now)

    private fun assertStep(step: Step, state: CompanionState, vararg effects: CompanionEffect) {
        assertEquals(state, step.model.state)
        assertEquals(effects.toList(), step.effects)
    }

    @Test
    fun pairingSendsThePairingRequestWithATimeLimit() {
        assertStep(
            stepOf(Stopped, PairWith(link)),
            Pairing,
            BeginPairing(link),
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
            stepOf(Pairing, PairingFailed(PairingProblem.APP_OUTDATED)),
            PairingFailure(PairingProblem.APP_OUTDATED),
            CancelPairingTimeout,
            EndHost,
        )
        assertStep(
            stepOf(Pairing, DeviceKeyUnavailable),
            PairingFailure(PairingProblem.DEVICE_KEY),
            CancelPairingTimeout,
            EndHost,
        )
    }

    @Test
    fun aPairingThatTimesOutGivesUpTheAttemptAndEndsTheHost() {
        assertStep(stepOf(Pairing, PairingTimedOut), PairingFailure(PairingProblem.PEER_UNREACHABLE), ForgetPairing, EndHost)
    }

    @Test
    fun anOutcomeOutsideAPairingChangesNothing() {
        assertStep(stepOf(Connected, PairingAccepted), Connected)
        assertStep(stepOf(Stopped, PairingFailed(PairingProblem.REFUSED)), Stopped)
    }

    @Test
    fun aRefusalOfThePeerOrAMissingDeviceKeyStaysDisconnectedAndTellsWhy() {
        for ((event, reason) in mapOf(
            ServicesEnded(ServicesEnding.DEVICE_REFUSED) to DisconnectReason.DEVICE_REFUSED,
            ServicesEnded(ServicesEnding.APP_OUTDATED) to DisconnectReason.APP_OUTDATED,
            ServicesEnded(ServicesEnding.PEER_OUTDATED) to DisconnectReason.PEER_OUTDATED,
            DeviceKeyUnavailable to DisconnectReason.DEVICE_KEY,
        )) {
            for (working in listOf(Connected, Reconnecting(lostPeer, 0), RetryingAt(lostPeer, now + FIRST_RETRY_MS, 1))) {
                assertStep(
                    stepOf(working, event),
                    Refused(reason),
                    StopServices,
                    CancelRetry,
                    PostFailureNotification(reason),
                    EndHost,
                )
            }
            assertStep(stepOf(Refused(reason), StartRequested), Refused(reason))
            assertStep(stepOf(Stopped, event), Stopped)
        }
    }

    @Test
    fun aMissingSigningKeyTriesAgainAfterAPause() {
        assertStep(
            stepOf(Connecting, SigningKeyUnavailable),
            RetryingAt(DisconnectReason.SIGNING_KEY, now + FIRST_RETRY_MS, 1),
            PostFailureNotification(DisconnectReason.SIGNING_KEY),
            ScheduleRetry(FIRST_RETRY_MS),
        )
    }

    @Test
    fun pairingAgainAfterAFailureStartsOver() {
        assertEquals(Pairing, stepOf(PairingFailure(PairingProblem.PEER_UNREACHABLE), PairWith(link)).model.state)
    }

    @Test
    fun cancellingPairingForgetsThePairLink() {
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
        assertStep(stepOf(Reconnecting(lostPeer, 1), unreachable, NetworkAccess.NONE), Offline(lostPeer, 2))
        assertStep(stepOf(RetryingAt(lostPeer, now, 2), NetworkChanged(NetworkAccess.NONE)), Offline(lostPeer, 2), CancelRetry)
        assertStep(stepOf(Offline(lostPeer, 2), usableNetwork, NetworkAccess.NONE), Reconnecting(lostPeer, 0), StartServices)
    }

    @Test
    fun withOnlyMeteredNetworksTheCompanionStopsAndWaitsForAnUnmeteredOne() {
        assertStep(stepOf(Connected, meteredNetworkOnly), WaitingForUnmeteredNetwork, StopServices, CancelRetry, CancelFailureNotification)
        assertStep(
            stepOf(RetryingAt(lostPeer, now, 2), meteredNetworkOnly),
            WaitingForUnmeteredNetwork,
            StopServices,
            CancelRetry,
            CancelFailureNotification,
        )
        assertStep(stepOf(Stopped, StartRequested, NetworkAccess.METERED_ONLY), WaitingForUnmeteredNetwork)
        assertStep(stepOf(Reconnecting(lostPeer, 1), unreachable, NetworkAccess.METERED_ONLY), WaitingForUnmeteredNetwork)
        assertStep(stepOf(WaitingForUnmeteredNetwork, usableNetwork, NetworkAccess.METERED_ONLY), Connecting, StartServices)
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
        assertStep(stepOf(RetryingAt(lostPeer, now + 40_000L, 4), usableNetwork), Reconnecting(lostPeer, 0), CancelRetry, StartServices)
        assertStep(stepOf(Reconnecting(lostPeer, 4), usableNetwork), Reconnecting(lostPeer, 0), RestartServices)
        assertStep(stepOf(Connected, usableNetwork), Connected, RestartServices)
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
        for (running in listOf(
            Connecting,
            Connected,
            Reconnecting(lostPeer, 1),
            RetryingAt(lostPeer, now, 2),
            Offline(lostPeer, 2),
            WaitingForUnmeteredNetwork,
        )) {
            assertStep(stepOf(running, batteryOnly), WaitingForCharger, StopServices, CancelRetry, CancelFailureNotification)
        }
        assertStep(stepOf(Stopped, StartRequested, powerAccess = PowerAccess.BATTERY_ONLY), WaitingForCharger)
    }

    @Test
    fun aChargerStartsAWaitingCompanion() {
        assertStep(stepOf(WaitingForCharger, usablePower, powerAccess = PowerAccess.BATTERY_ONLY), Connecting, StartServices)
        assertStep(stepOf(WaitingForCharger, usablePower, NetworkAccess.METERED_ONLY), WaitingForUnmeteredNetwork)
    }

    @Test
    fun aNetworkChangeLeavesACompanionWaitingForACharger() {
        for (networkAccess in NetworkAccess.entries) {
            assertStep(
                stepOf(WaitingForCharger, NetworkChanged(networkAccess), networkAccess, PowerAccess.BATTERY_ONLY),
                WaitingForCharger,
            )
        }
    }

    @Test
    fun pairingWithoutAChargerCompletesAndThenWaits() {
        assertStep(
            stepOf(Pairing, PairingAccepted, powerAccess = PowerAccess.BATTERY_ONLY),
            WaitingForCharger,
            CancelPairingTimeout,
            CompletePairing,
        )
    }

    @Test
    fun aChargerChangeLeavesACompanionAloneThatDoesNotRun() {
        for (resting in listOf(Pairing, Stopped, Refused(DisconnectReason.DEVICE_REFUSED), PairingFailure(PairingProblem.REFUSED))) {
            assertStep(stepOf(resting, batteryOnly), resting)
        }
        assertStep(stepOf(Connected, usablePower), Connected)
    }

    @Test
    fun theDeviceStaysAwakeOnlyWhileTheCompanionPairsOrRuns() {
        val awake = listOf(Pairing, Connecting, Connected, Reconnecting(lostPeer, 0), Offline(lostPeer, 1))
        val asleep = listOf(Stopped, WaitingForCharger, WaitingForUnmeteredNetwork, Refused(DisconnectReason.DEVICE_REFUSED))
        awake.forEach { assertEquals("$it keeps the device awake", true, it.keepsDeviceAwake()) }
        asleep.forEach { assertEquals("$it lets the device sleep", false, it.keepsDeviceAwake()) }
    }
}
