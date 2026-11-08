package io.github.nikitakarpei.yacypeercompanion.model

import org.junit.Assert.assertEquals
import org.junit.Test

class PowerConditionsTest {
    @Test
    fun aMissingChargerHoldsTheCompanionBackFirst() {
        assertEquals(PowerAccess.BATTERY_ONLY, PowerConditions(true, false, true, true).access)
    }

    @Test
    fun batterySaverHoldsTheCompanionBackOnlyWhileTheUserRespectsIt() {
        assertEquals(PowerAccess.BATTERY_SAVER_ON, PowerConditions(true, true, true, true).access)
        assertEquals(PowerAccess.USABLE, PowerConditions(true, true, false, true).access)
    }

    @Test
    fun withoutTheChargerRuleTheBatteryDoes() {
        assertEquals(PowerAccess.USABLE, PowerConditions(false, false, true, false).access)
        assertEquals(PowerAccess.BATTERY_SAVER_ON, PowerConditions(false, false, true, true).access)
    }
}
