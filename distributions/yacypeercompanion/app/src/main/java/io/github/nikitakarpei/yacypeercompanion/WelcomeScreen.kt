package io.github.nikitakarpei.yacypeercompanion

import android.app.Activity
import android.content.ActivityNotFoundException
import android.content.Intent
import android.net.Uri
import android.os.Bundle
import android.view.Gravity
import android.widget.LinearLayout

private const val SOURCE_URL = "https://github.com/nikitakarpei/my-yacy-node"
private const val LICENSE_URL = "https://www.gnu.org/licenses/agpl-3.0.html"
private const val ABOUT_TEXT =
    "YaCy Companion fetches web pages for a YaCy peer and builds their index entries. A key in the secure " +
        "hardware of this device signs each fetched page, so other peers can check that the page is genuine."

class WelcomeScreen : Activity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        val column = firstRunColumn(1, "Welcome")
        val rows = LinearLayout(this).apply { orientation = LinearLayout.VERTICAL }
        rows.addView(clickable(row(R.drawable.ic_info, "About", "What the app does")) { showAbout() })
        rows.addView(
            clickable(row(R.drawable.ic_code, "Source code", SOURCE_URL.removePrefix("https://"))) {
                open("Source code", SOURCE_URL)
            },
        )
        rows.addView(
            clickable(row(R.drawable.ic_license, "License", "GNU Affero General Public License v3.0")) {
                open("License", LICENSE_URL)
            },
        )
        column.addView(
            rows,
            LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                topMargin =
                    dp(24)
            },
        )
        column.spacer()
        column.addView(
            filledButton("Next") { startActivity(Intent(this, PermissionsScreen::class.java)) },
            LinearLayout.LayoutParams(dp(168), LinearLayout.LayoutParams.WRAP_CONTENT).apply { gravity = Gravity.END },
        )
        showScreen(column)
    }

    private fun clickable(row: LinearLayout, onClick: () -> Unit) = row.apply {
        isClickable = true
        isFocusable = true
        foreground = rowRipple()
        setOnClickListener { onClick() }
    }

    private fun showAbout() {
        showDialog("About", listOf(DialogAction("OK")), ABOUT_TEXT)
    }

    private fun open(title: String, url: String) {
        try {
            startActivity(Intent(Intent.ACTION_VIEW, Uri.parse(url)))
        } catch (_: ActivityNotFoundException) {
            showDialog(title, listOf(DialogAction("OK")), url)
        }
    }
}
