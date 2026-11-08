package io.github.nikitakarpei.yacypeercompanion

import android.app.Activity
import android.content.Intent
import android.os.Bundle
import android.view.Gravity
import android.widget.LinearLayout
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState

class SettingsScreen : Activity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val column = column().apply { setPadding(0, dp(12), 0, dp(24)) }
        column.addView(topBar("Settings", R.color.on_surface) { finish() })
        column.addView(
            chargerRow(),
            LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                topMargin = dp(16)
            },
        )
        column.addView(
            meteredNetworksRow(),
            LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT),
        )
        column.spacer()
        column.addView(
            destructiveButton("Unpair") { confirmUnpair() },
            LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                setMargins(dp(24), 0, dp(24), 0)
            },
        )
        showScreen(column)
    }

    override fun onResume() {
        super.onResume()
        CompanionStore.listener = { state -> if (state == CompanionState.Stopped) finishAndRemoveTask() }
    }

    override fun onPause() {
        CompanionStore.listener = null
        super.onPause()
    }

    private fun chargerRow() = LinearLayout(this).apply {
        orientation = LinearLayout.HORIZONTAL
        gravity = Gravity.CENTER_VERTICAL
        minimumHeight = dp(72)
        setPadding(dp(24), dp(8), dp(24), dp(8))
        val toggle =
            ToggleSwitch(context).apply {
                isChecked = Charger.required(context)
                setOnCheckedChangeListener { _, checked -> Charger.require(context, checked) }
            }
        addView(heading("Run only when charging", 16f), LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f))
        addView(toggle)
        isClickable = true
        foreground = rowRipple()
        setOnClickListener { toggle.toggle() }
    }

    private fun meteredNetworksRow() = LinearLayout(this).apply {
        orientation = LinearLayout.HORIZONTAL
        gravity = Gravity.CENTER_VERTICAL
        minimumHeight = dp(72)
        setPadding(dp(24), dp(8), dp(24), dp(8))
        val toggle =
            ToggleSwitch(context).apply {
                isChecked = MeteredNetworks.allowed(context)
                setOnCheckedChangeListener { _, checked -> MeteredNetworks.allow(context, checked) }
            }
        addView(
            LinearLayout(context).apply {
                orientation = LinearLayout.VERTICAL
                addView(heading("Use mobile data", 16f))
                addView(bodyText("Includes metered Wi-Fi", 14f))
            },
            LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f),
        )
        addView(toggle)
        isClickable = true
        foreground = rowRipple()
        setOnClickListener { toggle.toggle() }
    }

    private fun confirmUnpair() {
        showDialog(
            "Unpair this device?",
            listOf(DialogAction("Cancel"), DialogAction("Unpair", R.color.error) { unpair() }),
        )
    }

    private fun unpair() {
        CompanionStore.listener = null
        send(CompanionEvent.Unpair)
        startActivity(Intent(this, ScanScreen::class.java).addFlags(Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TASK))
        finish()
    }
}
