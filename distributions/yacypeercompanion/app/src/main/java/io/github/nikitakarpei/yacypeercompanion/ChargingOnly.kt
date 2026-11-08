package io.github.nikitakarpei.yacypeercompanion

import android.content.Context
import android.content.SharedPreferences

private const val PREFERENCES = "charging-only"
private const val REQUIRED_KEY = "required"

object ChargingOnly {
    fun required(ctx: Context): Boolean = preferencesOf(ctx).getBoolean(REQUIRED_KEY, true)

    fun require(ctx: Context, required: Boolean) {
        preferencesOf(ctx).edit().putBoolean(REQUIRED_KEY, required).apply()
    }

    fun follow(ctx: Context, listener: SharedPreferences.OnSharedPreferenceChangeListener) {
        preferencesOf(ctx).registerOnSharedPreferenceChangeListener(listener)
    }

    fun unfollow(ctx: Context, listener: SharedPreferences.OnSharedPreferenceChangeListener) {
        preferencesOf(ctx).unregisterOnSharedPreferenceChangeListener(listener)
    }

    private fun preferencesOf(ctx: Context) = ctx.getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE)
}
