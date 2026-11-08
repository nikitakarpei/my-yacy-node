package io.github.nikitakarpei.yacypeercompanion

import android.content.Context
import android.content.SharedPreferences

private const val PREFERENCES = "battery-saver-pause"
private const val WANTED_KEY = "wanted"

object BatterySaverPause {
    fun wanted(ctx: Context): Boolean = preferencesOf(ctx).getBoolean(WANTED_KEY, true)

    fun want(ctx: Context, wanted: Boolean) {
        preferencesOf(ctx).edit().putBoolean(WANTED_KEY, wanted).apply()
    }

    fun follow(ctx: Context, listener: SharedPreferences.OnSharedPreferenceChangeListener) {
        preferencesOf(ctx).registerOnSharedPreferenceChangeListener(listener)
    }

    fun unfollow(ctx: Context, listener: SharedPreferences.OnSharedPreferenceChangeListener) {
        preferencesOf(ctx).unregisterOnSharedPreferenceChangeListener(listener)
    }

    private fun preferencesOf(ctx: Context) = ctx.getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE)
}
