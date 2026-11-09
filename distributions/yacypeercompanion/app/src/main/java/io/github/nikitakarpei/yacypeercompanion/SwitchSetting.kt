package io.github.nikitakarpei.yacypeercompanion

import android.content.Context
import android.content.SharedPreferences

class SwitchSetting(private val file: String, private val key: String, private val default: Boolean) {
    fun isOn(ctx: Context): Boolean = preferencesOf(ctx).getBoolean(key, default)

    fun set(ctx: Context, on: Boolean) {
        preferencesOf(ctx).edit().putBoolean(key, on).apply()
    }

    fun follow(ctx: Context, listener: SharedPreferences.OnSharedPreferenceChangeListener) {
        preferencesOf(ctx).registerOnSharedPreferenceChangeListener(listener)
    }

    fun unfollow(ctx: Context, listener: SharedPreferences.OnSharedPreferenceChangeListener) {
        preferencesOf(ctx).unregisterOnSharedPreferenceChangeListener(listener)
    }

    private fun preferencesOf(ctx: Context) = ctx.getSharedPreferences(file, Context.MODE_PRIVATE)
}

val chargerRequired = SwitchSetting("charger", "required", true)

val meteredNetworksAllowed = SwitchSetting("metered-networks", "allowed", false)

val batterySaverRespected = SwitchSetting("battery-saver", "respected", true)
