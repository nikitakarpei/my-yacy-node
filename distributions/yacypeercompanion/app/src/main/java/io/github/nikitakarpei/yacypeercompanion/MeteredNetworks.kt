package io.github.nikitakarpei.yacypeercompanion

import android.content.Context
import android.content.SharedPreferences

private const val PREFERENCES = "metered-networks"
private const val ALLOWED_KEY = "allowed"

object MeteredNetworks {
    fun allowed(ctx: Context): Boolean = preferencesOf(ctx).getBoolean(ALLOWED_KEY, false)

    fun allow(ctx: Context, allowed: Boolean) {
        preferencesOf(ctx).edit().putBoolean(ALLOWED_KEY, allowed).apply()
    }

    fun follow(ctx: Context, listener: SharedPreferences.OnSharedPreferenceChangeListener) {
        preferencesOf(ctx).registerOnSharedPreferenceChangeListener(listener)
    }

    fun unfollow(ctx: Context, listener: SharedPreferences.OnSharedPreferenceChangeListener) {
        preferencesOf(ctx).unregisterOnSharedPreferenceChangeListener(listener)
    }

    private fun preferencesOf(ctx: Context) = ctx.getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE)
}
