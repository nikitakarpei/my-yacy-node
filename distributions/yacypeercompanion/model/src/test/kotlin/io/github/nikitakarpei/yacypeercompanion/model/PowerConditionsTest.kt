package io.github.nikitakarpei.yacypeercompanion.model

import org.junit.Assert.assertEquals
import org.junit.Test

class PowerConditionsTest {
    @Test
    fun aMissingChargerHoldsTheCompanionBackFirst() {
        assertEquals(
            PowerAccess.BATTERY_ONLY,
            PowerConditions(chargerRequired = true, onCharger = false, batterySaverRespected = true, batterySaverOn = true).access,
        )
    }

    @Test
    fun batterySaverHoldsTheCompanionBackOnlyWhileTheUserRespectsIt() {
        assertEquals(
            PowerAccess.BATTERY_SAVER_ON,
            PowerConditions(chargerRequired = true, onCharger = true, batterySaverRespected = true, batterySaverOn = true).access,
        )
        assertEquals(
            PowerAccess.USABLE,
            PowerConditions(chargerRequired = true, onCharger = true, batterySaverRespected = false, batterySaverOn = true).access,
        )
    }

    @Test
    fun turningOffTheChargerRuleLetsTheCompanionRunOnBattery() {
        assertEquals(
            PowerAccess.USABLE,
            PowerConditions(chargerRequired = false, onCharger = false, batterySaverRespected = true, batterySaverOn = false).access,
        )
    }
}
