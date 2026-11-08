package io.github.nikitakarpei.yacypeercompanion

import android.content.Context

private const val PREFERENCES = "data-sharing"
private const val GIVEN_KEY = "consent-given"

object DataSharingConsent {
    fun given(ctx: Context): Boolean = preferencesOf(ctx).getBoolean(GIVEN_KEY, false)

    fun give(ctx: Context) {
        preferencesOf(ctx).edit().putBoolean(GIVEN_KEY, true).apply()
    }

    private fun preferencesOf(ctx: Context) = ctx.getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE)
}
