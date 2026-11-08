package io.github.nikitakarpei.yacypeercompanion

import android.app.Activity
import android.content.Intent
import android.graphics.drawable.GradientDrawable
import android.os.Bundle
import android.view.Gravity
import android.widget.LinearLayout
import io.github.nikitakarpei.yacypeercompanion.model.CompanionEvent
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState

class HomeScreen : Activity() {
    private lateinit var column: LinearLayout
    private val ticker = StatusTicker()

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        if (!Pairing.paired(this)) {
            val firstScreen = if (pairingAllowed(this)) ScanScreen::class.java else WelcomeScreen::class.java
            startActivity(Intent(this, firstScreen))
            finish()
            return
        }
        column = column().apply { setPadding(dp(24), dp(24), dp(24), dp(24)) }
        showScreen(column)
    }

    override fun onResume() {
        super.onResume()
        if (!::column.isInitialized) return
        if (CompanionStore.state == CompanionState.Stopped) send(CompanionEvent.StartRequested)
        CompanionStore.listener = { state -> closeOnStopOrDraw(state) }
        draw(CompanionStore.state)
    }

    override fun onPause() {
        CompanionStore.listener = null
        ticker.stop()
        super.onPause()
    }

    private fun closeOnStopOrDraw(state: CompanionState) {
        if (state == CompanionState.Stopped) finishAndRemoveTask() else draw(state)
    }

    private fun draw(state: CompanionState) {
        column.removeAllViews()
        ticker.stop()
        column.addView(appBar())
        for (permission in Permission.lasting.filter { !it.granted(this) }) {
            column.addView(
                permissionBanner(permission),
                LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                    topMargin =
                        dp(16)
                },
            )
        }
        column.spacer()
        drawStatus(state)
        column.spacer()
    }

    private fun appBar() = LinearLayout(this).apply {
        orientation = LinearLayout.HORIZONTAL
        gravity = Gravity.CENTER_VERTICAL
        addView(heading(APP_NAME, 22f), LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f))
        addView(
            iconButton(R.drawable.ic_settings, R.color.on_surface_variant, "Settings") {
                startActivity(Intent(this@HomeScreen, SettingsScreen::class.java))
            }.apply { (layoutParams as LinearLayout.LayoutParams).marginEnd = -dp(12) },
        )
    }

    private fun drawStatus(state: CompanionState) {
        val badge =
            when (state) {
                is CompanionState.Disconnected -> statusBadge(R.drawable.ic_alert, R.color.error_container, R.color.on_error_container)

                CompanionState.Connected -> statusBadge(R.drawable.ic_check, R.color.primary_container, R.color.on_primary_container)

                is CompanionState.Waiting -> statusBadge(
                    waitingIconOf(state),
                    R.color.surface_container_highest,
                    R.color.on_surface_variant,
                )

                else -> ProgressRing(this)
            }
        column.addView(badge, LinearLayout.LayoutParams(dp(120), dp(120)).apply { gravity = Gravity.CENTER_HORIZONTAL })
        column.addView(centered(heading(titleOf(state), 32f)).apply { setPadding(0, dp(24), 0, dp(8)) })
        if (state is CompanionState.Disconnected) {
            column.addView(centered(bodyText(reasonTextOf(state))))
            if (attemptTextOf(state) != null) {
                val attemptLine = centered(bodyText("", 14f)).apply { setPadding(0, dp(8), 0, 0) }
                column.addView(attemptLine)
                ticker.follow(state, attemptLine)
            }
        }
    }

    private fun centered(view: android.widget.TextView) = view.apply {
        gravity = Gravity.CENTER_HORIZONTAL
        layoutParams = LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT)
    }

    private fun permissionBanner(permission: Permission) = LinearLayout(this).apply {
        orientation = LinearLayout.HORIZONTAL
        gravity = Gravity.CENTER_VERTICAL
        setPadding(dp(16), dp(12), dp(8), dp(12))
        background =
            GradientDrawable().apply {
                cornerRadius = dp(16).toFloat()
                setColor(colorOf(R.color.error_container))
            }
        addView(
            icon(
                if (permission ==
                    Permission.NOTIFICATIONS
                ) {
                    R.drawable.ic_bell_off
                } else {
                    R.drawable.ic_battery
                },
                R.color.on_error_container,
                24,
            ),
        )
        addView(
            bodyText(bannerTextOf(permission), 14f, R.color.on_error_container),
            LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply { marginStart = dp(12) },
        )
        addView(textButton("Turn on", R.color.on_error_container) { permission.ask(this@HomeScreen) })
    }

    private fun bannerTextOf(permission: Permission) = when (permission) {
        Permission.NOTIFICATIONS -> "Notifications are off, so status and problems stay hidden."
        else -> "Running in the background is off, so Android can stop the app with the screen off."
    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<out String>, results: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, results)
        Permission.noteAnswers(this, requestCode, results)
    }
}
