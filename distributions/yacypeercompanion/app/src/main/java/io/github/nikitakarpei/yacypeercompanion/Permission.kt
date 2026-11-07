package io.github.nikitakarpei.yacypeercompanion

import android.Manifest
import android.annotation.SuppressLint
import android.app.Activity
import android.app.NotificationManager
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.net.Uri
import android.os.Build
import android.os.PowerManager
import android.provider.Settings

private const val PREFERENCES = "permissions"
private const val REFUSED_SUFFIX = ".refused"

enum class Permission {
    CAMERA,
    NOTIFICATIONS,
    BACKGROUND,
    ;

    fun granted(ctx: Context): Boolean = when (this) {
        CAMERA -> ctx.checkSelfPermission(Manifest.permission.CAMERA) == PackageManager.PERMISSION_GRANTED
        NOTIFICATIONS -> ctx.getSystemService(NotificationManager::class.java).areNotificationsEnabled()
        BACKGROUND -> ctx.getSystemService(PowerManager::class.java).isIgnoringBatteryOptimizations(ctx.packageName)
    }

    fun ask(activity: Activity) {
        val runtime = runtimePermission() ?: return openSettings(activity)
        if (refusedForGood(activity)) return openSettings(activity)
        activity.requestPermissions(arrayOf(runtime), ordinal)
    }

    fun refusedForGood(ctx: Context): Boolean = preferencesOf(ctx).getBoolean(name + REFUSED_SUFFIX, false)

    fun noteAnswer(activity: Activity, granted: Boolean) {
        val runtime = runtimePermission() ?: return
        val refusedForGood = !granted && !activity.shouldShowRequestPermissionRationale(runtime)
        preferencesOf(activity).edit().putBoolean(name + REFUSED_SUFFIX, refusedForGood).apply()
    }

    private fun runtimePermission(): String? = when {
        this == CAMERA -> Manifest.permission.CAMERA
        this == NOTIFICATIONS && Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU -> Manifest.permission.POST_NOTIFICATIONS
        else -> null
    }

    @SuppressLint("BatteryLife")
    private fun openSettings(activity: Activity) {
        val intent =
            when (this) {
                BACKGROUND -> Intent(Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS, Uri.parse("package:${activity.packageName}"))

                NOTIFICATIONS -> Intent(
                    Settings.ACTION_APP_NOTIFICATION_SETTINGS,
                ).putExtra(Settings.EXTRA_APP_PACKAGE, activity.packageName)

                CAMERA -> Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS, Uri.parse("package:${activity.packageName}"))
            }
        activity.startActivity(intent)
    }

    companion object {
        val lasting = listOf(NOTIFICATIONS, BACKGROUND)

        fun lastingGranted(ctx: Context): Boolean = lasting.all { it.granted(ctx) }

        fun noteAnswers(activity: Activity, requestCode: Int, results: IntArray) {
            val permission = entries.getOrNull(requestCode) ?: return
            permission.noteAnswer(activity, results.firstOrNull() == PackageManager.PERMISSION_GRANTED)
        }
    }
}

private fun preferencesOf(ctx: Context) = ctx.getSharedPreferences(PREFERENCES, Context.MODE_PRIVATE)
