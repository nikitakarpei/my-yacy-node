package io.github.nikitakarpei.yacypeercompanion

import android.app.Activity
import android.app.Dialog
import android.graphics.Color
import android.graphics.drawable.ColorDrawable
import android.graphics.drawable.GradientDrawable
import android.view.Gravity
import android.view.View
import android.view.Window
import android.widget.LinearLayout
import android.widget.ScrollView

private const val SCRIM_ALPHA = 0.32f

class DialogAction(val text: String, val colorId: Int = R.color.primary, val onClick: () -> Unit = {})

fun Activity.showDialog(
    title: String,
    actions: List<DialogAction>,
    message: String? = null,
    iconId: Int? = null,
    content: View? = null,
    onCancel: () -> Unit = {},
) {
    val dialog = Dialog(this)
    val box =
        LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            setPadding(dp(24), dp(24), dp(24), dp(16))
            background =
                GradientDrawable().apply {
                    cornerRadius = dp(28).toFloat()
                    setColor(colorOf(R.color.surface_container_high))
                }
        }
    iconId?.let {
        box.addView(icon(it, R.color.error, 24), LinearLayout.LayoutParams(dp(24), dp(24)).apply { gravity = Gravity.CENTER_HORIZONTAL })
    }
    box.addView(
        heading(title, 24f).apply { if (iconId != null) gravity = Gravity.CENTER_HORIZONTAL },
        LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
            topMargin = if (iconId != null) dp(16) else 0
        },
    )
    message?.let {
        box.addView(
            bodyText(it, 14f),
            LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                topMargin = dp(16)
            },
        )
    }
    content?.let {
        box.addView(
            ScrollView(this).apply { addView(it) },
            LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply {
                topMargin = dp(16)
            },
        )
    }
    val buttons = LinearLayout(this).apply { gravity = Gravity.END }
    for ((index, action) in actions.withIndex()) {
        buttons.addView(
            textButton(action.text, action.colorId) {
                dialog.dismiss()
                action.onClick()
            }.apply {
                textSize = 14f
                minHeight = dp(40)
                minimumHeight = dp(40)
                minWidth = dp(48)
                minimumWidth = dp(48)
                setPadding(dp(12), 0, dp(12), 0)
            },
            LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
                marginStart = if (index == 0) 0 else dp(8)
            },
        )
    }
    box.addView(
        buttons,
        LinearLayout.LayoutParams(LinearLayout.LayoutParams.MATCH_PARENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
            topMargin = dp(16)
        },
    )
    dialog.requestWindowFeature(Window.FEATURE_NO_TITLE)
    dialog.setContentView(box)
    dialog.setOnCancelListener { onCancel() }
    dialog.show()
    dialog.window?.apply {
        setBackgroundDrawable(ColorDrawable(Color.TRANSPARENT))
        setDimAmount(SCRIM_ALPHA)
        setLayout((resources.displayMetrics.widthPixels - dp(48)).coerceAtMost(dp(560)), LinearLayout.LayoutParams.WRAP_CONTENT)
    }
}
