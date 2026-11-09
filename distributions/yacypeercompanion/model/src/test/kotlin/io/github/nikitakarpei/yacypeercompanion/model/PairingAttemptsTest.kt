package io.github.nikitakarpei.yacypeercompanion.model

import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.PairingAccepted
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent.PairingFailed
import io.github.nikitakarpei.yacypeercompanion.model.KeyDisposal.DeleteKey
import io.github.nikitakarpei.yacypeercompanion.model.KeyDisposal.KeepKey
import io.github.nikitakarpei.yacypeercompanion.model.KeyDisposal.UnpairAndDeleteKey
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Test

class PairingAttemptsTest {
    private val link = "yacypeer-companion://pair?address=192.168.1.20%3A4223&brokerKeyHash=hash&secret=secret"
    private val refused = PairingFailed(PairingProblem.REFUSED)

    @Test
    fun theOutcomeOfARunningAttemptGoesToTheApp() {
        val attempts = PairingAttempts()

        assertNull(attempts.ended(attempts.started(link), PairingAccepted))
    }

    @Test
    fun anAcceptanceAfterTheAppGaveUpUnpairsTheDevice() {
        val attempts = PairingAttempts()
        val attempt = attempts.started(link)
        assertNull(attempts.givenUp())

        assertEquals(UnpairAndDeleteKey(link), attempts.ended(attempt, PairingAccepted))
    }

    @Test
    fun aFailureAfterTheAppGaveUpDeletesTheKey() {
        val attempts = PairingAttempts()
        val attempt = attempts.started(link)
        attempts.givenUp()

        assertEquals(DeleteKey, attempts.ended(attempt, refused))
    }

    @Test
    fun givingUpAfterAFailureDeletesTheKey() {
        val attempts = PairingAttempts()
        attempts.ended(attempts.started(link), refused)

        assertEquals(DeleteKey, attempts.givenUp())
    }

    @Test
    fun givingUpWithoutAnAttemptDeletesTheKey() {
        assertEquals(DeleteKey, PairingAttempts().givenUp())
    }

    @Test
    fun aNewerAttemptKeepsTheKeyWhenAnOlderOneEndsLate() {
        val attempts = PairingAttempts()
        val older = attempts.started(link)
        attempts.givenUp()
        attempts.started(link)

        assertEquals(KeepKey, attempts.ended(older, PairingAccepted))
    }
}
