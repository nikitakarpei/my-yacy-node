package io.github.nikitakarpei.yacypeercompanion

import android.content.Context

private const val PREFERENCES = "data-sharing"
private const val AGREED_KEY = "agreed"

object DataSharing {
    fun agreed(ctx: Context): Boolean = preferencesOf(ctx).getBoolean(AGREED_KEY, false)

    fun agree(ctx: Context) {
        preferencesOf(ctx).edit().putBoolean(AGREED_KEY, true).apply()
    }

    private fun preferencesOf(ctx: Context) = ctx.getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE)
}
