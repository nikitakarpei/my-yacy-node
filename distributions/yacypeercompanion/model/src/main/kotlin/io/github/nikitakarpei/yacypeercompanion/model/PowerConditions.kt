package io.github.nikitakarpei.yacypeercompanion.model

data class PowerConditions(
    val chargerRequired: Boolean,
    val onCharger: Boolean,
    val batterySaverRespected: Boolean,
    val batterySaverOn: Boolean,
) {
    val access: PowerAccess
        get() = when {
            chargerRequired && !onCharger -> PowerAccess.BATTERY_ONLY
            batterySaverRespected && batterySaverOn -> PowerAccess.BATTERY_SAVER_ON
            else -> PowerAccess.USABLE
        }
}
