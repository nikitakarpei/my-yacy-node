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
            switchRow("Run only when charging", chargerRequired),
            LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                topMargin = dp(16)
            },
        )
        column.addView(
            switchRow("Pause while Battery Saver is on", batterySaverRespected),
            LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT),
        )
        column.addView(
            switchRowWithDetail("Use mobile data", "Includes metered Wi-Fi", meteredNetworksAllowed),
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

    private fun switchRow(title: String, setting: SwitchSetting) = switchRowWith(setting) { texts -> texts.addView(heading(title, 16f)) }

    private fun switchRowWithDetail(title: String, detail: String, setting: SwitchSetting) = switchRowWith(setting) { texts ->
        texts.addView(heading(title, 16f))
        texts.addView(bodyText(detail, 14f))
    }

    private fun switchRowWith(setting: SwitchSetting, addTexts: (LinearLayout) -> Unit) = LinearLayout(this).apply {
        orientation = LinearLayout.HORIZONTAL
        gravity = Gravity.CENTER_VERTICAL
        minimumHeight = dp(72)
        setPadding(dp(24), dp(8), dp(24), dp(8))
        val toggle =
            ToggleSwitch(context).apply {
                isChecked = setting.isOn(context)
                setOnCheckedChangeListener { _, on -> setting.set(context, on) }
            }
        addView(
            LinearLayout(context).apply {
                orientation = LinearLayout.VERTICAL
                addTexts(this)
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
