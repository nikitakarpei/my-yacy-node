package io.github.nikitakarpei.yacypeercompanion.model

import org.junit.Assert.assertEquals
import org.junit.Test

class PairingRuleTest {
    @Test
    fun pairingWaitsForThePermissionsAndTheDataSharingAgreement() {
        assertEquals(true, pairingAllowed(permissionsGranted = true, dataSharingAgreed = true))
        assertEquals(false, pairingAllowed(permissionsGranted = true, dataSharingAgreed = false))
        assertEquals(false, pairingAllowed(permissionsGranted = false, dataSharingAgreed = true))
    }
}
