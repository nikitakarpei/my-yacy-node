package io.github.nikitakarpei.yacypeercompanion

import android.content.Context

private const val PREFERENCES = "pairing"
private const val LINK_KEY = "link"
private const val PAIRED_KEY = "paired"

object Pairing {
    fun linkText(ctx: Context): String? = preferences(ctx).getString(LINK_KEY, null)

    fun paired(ctx: Context): Boolean = preferences(ctx).getBoolean(PAIRED_KEY, false)

    fun begin(ctx: Context, linkText: String) {
        preferences(ctx).edit().putString(LINK_KEY, linkText).putBoolean(PAIRED_KEY, false).apply()
    }

    fun complete(ctx: Context) {
        preferences(ctx).edit().putBoolean(PAIRED_KEY, true).apply()
    }

    fun forget(ctx: Context) {
        preferences(ctx).edit().clear().apply()
    }

    private fun preferences(ctx: Context) = ctx.getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE)
}
