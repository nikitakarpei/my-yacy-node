package io.github.nikitakarpei.yacypeercompanion

import android.app.Activity
import android.content.Intent
import android.os.Bundle
import android.view.Gravity
import android.widget.Button
import android.widget.LinearLayout

class PermissionsScreen : Activity() {
    private lateinit var rows: LinearLayout
    private lateinit var next: Button

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val column = firstRunColumn(2, "Permissions")
        rows = LinearLayout(this).apply { orientation = LinearLayout.VERTICAL }
        column.addView(
            rows,
            LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                topMargin =
                    dp(24)
            },
        )
        column.spacer()
        next = filledButton("Next") { startActivity(Intent(this, ScanScreen::class.java)) }
        column.addView(
            LinearLayout(this).apply {
                orientation = LinearLayout.HORIZONTAL
                gravity = Gravity.CENTER_VERTICAL
                addView(textButton("Back") { finish() })
                spacer()
                addView(next, LinearLayout.LayoutParams(dp(168), LinearLayout.LayoutParams.WRAP_CONTENT))
            },
            LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT),
        )
        showScreen(column)
    }

    override fun onResume() {
        super.onResume()
        refresh()
    }

    override fun onRequestPermissionsResult(requestCode: Int, permissions: Array<out String>, results: IntArray) {
        super.onRequestPermissionsResult(requestCode, permissions, results)
        Permission.noteAnswers(this, requestCode, results)
        refresh()
    }

    private fun refresh() {
        rows.removeAllViews()
        for (permission in Permission.lasting) {
            val end = if (permission.granted(this)) doneMark("Allowed") else tonalButton("Allow") { permission.ask(this) }
            rows.addView(rowOf(permission, end))
        }
        rows.addView(dataSharingRow())
        next.isEnabled = PairingPrerequisites.met(this)
    }

    private fun dataSharingRow(): LinearLayout {
        val end = if (DataSharing.agreed(this)) {
            doneMark("Agreed")
        } else {
            tonalButton("Review") { showDataSharingTerms { agreeToDataSharing() } }
        }
        return titleRow(R.drawable.ic_shield, DATA_SHARING_TITLE, end)
    }

    private fun agreeToDataSharing() {
        DataSharing.agree(this)
        refresh()
    }

    private fun rowOf(permission: Permission, end: android.view.View) = when (permission) {
        Permission.NOTIFICATIONS -> row(R.drawable.ic_bell, "Notifications", "For showing status and problems", end)
        else -> row(R.drawable.ic_battery, "Run in background", "For working with the screen off", end)
    }

    private fun doneMark(text: String) = LinearLayout(this).apply {
        orientation = LinearLayout.HORIZONTAL
        gravity = Gravity.CENTER_VERTICAL
        addView(icon(R.drawable.ic_check, R.color.primary, 20))
        addView(bodyText(text, 14f, R.color.primary).apply { setPadding(dp(4), 0, 0, 0) })
    }
}
