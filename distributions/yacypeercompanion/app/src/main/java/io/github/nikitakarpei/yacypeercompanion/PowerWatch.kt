package io.github.nikitakarpei.yacypeercompanion

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.SharedPreferences
import android.os.BatteryManager
import android.util.Log
import io.github.nikitakarpei.yacypeercompanion.model.PowerAccess

private const val LOG_TAG = "PowerWatch"

class PowerWatch(private val ctx: Context, private val onChange: (PowerAccess) -> Unit) : AutoCloseable {
    @Volatile
    private var onCharger = onChargerNow()

    @Volatile
    private var reportedAccess: PowerAccess? = null

    val access: PowerAccess
        get() = if (!Charger.required(ctx) || onCharger) PowerAccess.USABLE else PowerAccess.BATTERY_ONLY

    private val chargerReceiver =
        object : BroadcastReceiver() {
            override fun onReceive(context: Context, intent: Intent) {
                onCharger = intent.action == Intent.ACTION_POWER_CONNECTED
                report(intent.action.orEmpty())
            }
        }

    private val settingListener = SharedPreferences.OnSharedPreferenceChangeListener { _, _ -> report("charger setting") }

    init {
        val chargerChanges =
            IntentFilter().apply {
                addAction(Intent.ACTION_POWER_CONNECTED)
                addAction(Intent.ACTION_POWER_DISCONNECTED)
            }
        ctx.registerReceiver(chargerReceiver, chargerChanges, Context.RECEIVER_NOT_EXPORTED)
        Charger.follow(ctx, settingListener)
        reportedAccess = access
    }

    private fun onChargerNow(): Boolean {
        val battery = ctx.registerReceiver(null, IntentFilter(Intent.ACTION_BATTERY_CHANGED))
        if (battery == null) {
            Log.w(LOG_TAG, "battery state unknown, so the device counts as off the charger")
            return false
        }
        return battery.getIntExtra(BatteryManager.EXTRA_PLUGGED, 0) != 0
    }

    private fun report(change: String) {
        val access = access
        if (access == reportedAccess) return
        reportedAccess = access
        Log.d(LOG_TAG, "power $change, access: $access")
        onChange(access)
    }

    override fun close() {
        Charger.unfollow(ctx, settingListener)
        ctx.unregisterReceiver(chargerReceiver)
    }
}
