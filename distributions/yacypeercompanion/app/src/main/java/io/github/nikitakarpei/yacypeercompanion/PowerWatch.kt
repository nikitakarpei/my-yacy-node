package io.github.nikitakarpei.yacypeercompanion

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.SharedPreferences
import android.os.BatteryManager
import android.os.PowerManager
import android.util.Log
import io.github.nikitakarpei.yacypeercompanion.model.PowerAccess

private const val LOG_TAG = "PowerWatch"

class PowerWatch(private val ctx: Context, private val onChange: (PowerAccess) -> Unit) : AutoCloseable {
    private val powerManager = ctx.getSystemService(PowerManager::class.java)

    @Volatile
    private var pluggedIn = pluggedInNow()

    @Volatile
    private var reportedAccess: PowerAccess? = null

    val access: PowerAccess
        get() = when {
            ChargingOnly.required(ctx) && !pluggedIn -> PowerAccess.CHARGER_NEEDED
            BatterySaverPause.wanted(ctx) && powerManager.isPowerSaveMode -> PowerAccess.BATTERY_SAVER_ON
            else -> PowerAccess.USABLE
        }

    private val powerChanges =
        object : BroadcastReceiver() {
            override fun onReceive(context: Context, intent: Intent) {
                when (intent.action) {
                    Intent.ACTION_POWER_CONNECTED -> pluggedIn = true
                    Intent.ACTION_POWER_DISCONNECTED -> pluggedIn = false
                }
                report(intent.action.orEmpty())
            }
        }

    private val powerSettings = SharedPreferences.OnSharedPreferenceChangeListener { _, _ -> report("power setting") }

    init {
        val changes =
            IntentFilter().apply {
                addAction(Intent.ACTION_POWER_CONNECTED)
                addAction(Intent.ACTION_POWER_DISCONNECTED)
                addAction(PowerManager.ACTION_POWER_SAVE_MODE_CHANGED)
            }
        ctx.registerReceiver(powerChanges, changes, Context.RECEIVER_NOT_EXPORTED)
        ChargingOnly.follow(ctx, powerSettings)
        BatterySaverPause.follow(ctx, powerSettings)
        reportedAccess = access
    }

    private fun pluggedInNow(): Boolean {
        val battery = ctx.registerReceiver(null, IntentFilter(Intent.ACTION_BATTERY_CHANGED))
        return (battery?.getIntExtra(BatteryManager.EXTRA_PLUGGED, 0) ?: 0) != 0
    }

    private fun report(change: String) {
        val access = access
        if (access == reportedAccess) return
        reportedAccess = access
        Log.d(LOG_TAG, "power $change, access: $access")
        onChange(access)
    }

    override fun close() {
        BatterySaverPause.unfollow(ctx, powerSettings)
        ChargingOnly.unfollow(ctx, powerSettings)
        ctx.unregisterReceiver(powerChanges)
    }
}
