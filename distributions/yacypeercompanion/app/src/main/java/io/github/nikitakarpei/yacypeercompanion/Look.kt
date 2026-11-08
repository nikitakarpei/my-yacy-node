package io.github.nikitakarpei.yacypeercompanion

import android.app.Activity
import android.content.Context
import android.content.res.ColorStateList
import android.graphics.Typeface
import android.graphics.drawable.ColorDrawable
import android.graphics.drawable.GradientDrawable
import android.graphics.drawable.RippleDrawable
import android.view.Gravity
import android.view.View
import android.widget.Button
import android.widget.FrameLayout
import android.widget.ImageButton
import android.widget.ImageView
import android.widget.LinearLayout
import android.widget.TextView
import io.github.nikitakarpei.yacypeercompanion.model.CompanionState
import io.github.nikitakarpei.yacypeercompanion.model.Step
import kotlin.math.roundToInt

fun Context.dp(value: Int): Int = (value * resources.displayMetrics.density).roundToInt()

fun Context.colorOf(id: Int): Int = getColor(id)

fun Context.column(): LinearLayout = LinearLayout(this).apply {
    orientation = LinearLayout.VERTICAL
    setBackgroundColor(colorOf(R.color.surface))
}

fun Context.heading(text: String, size: Float = 36f): TextView = TextView(this).apply {
    this.text = text
    textSize = size
    setTextColor(colorOf(R.color.on_surface))
}

fun Context.bodyText(text: String, size: Float = 16f, colorId: Int = R.color.on_surface_variant): TextView = TextView(this).apply {
    this.text = text
    textSize = size
    setTextColor(colorOf(colorId))
}

private fun Context.pill(color: ColorStateList): GradientDrawable = GradientDrawable().apply {
    cornerRadius = dp(26).toFloat()
    this.color = color
}

private fun Context.basicButton(text: String, onClick: () -> Unit): Button = Button(this).apply {
    this.text = text
    isAllCaps = false
    textSize = 16f
    typeface = Typeface.create("sans-serif-medium", Typeface.NORMAL)
    stateListAnimator = null
    minHeight = dp(52)
    minimumHeight = dp(52)
    setPadding(dp(24), 0, dp(24), 0)
    setOnClickListener { onClick() }
}

private val enabledStates = arrayOf(intArrayOf(-android.R.attr.state_enabled), intArrayOf())

fun Context.filledButton(text: String, onClick: () -> Unit): Button = basicButton(text, onClick).apply {
    val background = ColorStateList(enabledStates, intArrayOf(colorOf(R.color.disabled), colorOf(R.color.primary)))
    this.background = RippleDrawable(ColorStateList.valueOf(colorOf(R.color.on_primary)), pill(background), null)
    setTextColor(ColorStateList(enabledStates, intArrayOf(colorOf(R.color.on_disabled), colorOf(R.color.on_primary))))
}

fun Context.tonalButton(text: String, onClick: () -> Unit): Button = basicButton(text, onClick).apply {
    textSize = 14f
    minHeight = dp(40)
    minimumHeight = dp(40)
    setPadding(dp(20), 0, dp(20), 0)
    background =
        RippleDrawable(
            ColorStateList.valueOf(colorOf(R.color.on_primary_container)),
            pill(ColorStateList.valueOf(colorOf(R.color.primary_container))),
            null,
        )
    setTextColor(colorOf(R.color.on_primary_container))
}

fun Context.textButton(text: String, colorId: Int = R.color.primary, onClick: () -> Unit): Button = basicButton(text, onClick).apply {
    background =
        RippleDrawable(ColorStateList.valueOf(colorOf(R.color.primary_container)), null, pill(ColorStateList.valueOf(0xFFFFFFFF.toInt())))
    setTextColor(colorOf(colorId))
}

fun Context.destructiveButton(text: String, onClick: () -> Unit): Button = basicButton(text, onClick).apply {
    background =
        RippleDrawable(
            ColorStateList.valueOf(colorOf(R.color.error)),
            pill(ColorStateList.valueOf(colorOf(R.color.error_container))),
            null,
        )
    setTextColor(colorOf(R.color.error))
}

fun Context.topBar(title: String, colorId: Int, onBack: () -> Unit): LinearLayout = LinearLayout(this).apply {
    orientation = LinearLayout.HORIZONTAL
    gravity = Gravity.CENTER_VERTICAL
    setPadding(dp(8), dp(8), dp(8), dp(8))
    addView(iconButton(R.drawable.ic_back, colorId, "Back", onBack))
    addView(heading(title, 22f).apply { setTextColor(colorOf(colorId)) })
}

fun Context.iconButton(drawableId: Int, colorId: Int, description: String, onClick: () -> Unit): ImageButton = ImageButton(this).apply {
    setImageResource(drawableId)
    imageTintList = ColorStateList.valueOf(colorOf(colorId))
    background = RippleDrawable(ColorStateList.valueOf(colorOf(R.color.outline_variant)), null, null)
    contentDescription = description
    setOnClickListener { onClick() }
    layoutParams = LinearLayout.LayoutParams(dp(48), dp(48))
}

fun Context.rowRipple(): RippleDrawable =
    RippleDrawable(ColorStateList.valueOf(colorOf(R.color.outline_variant)), null, ColorDrawable(colorOf(R.color.on_surface)))

fun Context.icon(drawableId: Int, colorId: Int, size: Int = 26): ImageView = ImageView(this).apply {
    setImageResource(drawableId)
    imageTintList = ColorStateList.valueOf(colorOf(colorId))
    layoutParams = LinearLayout.LayoutParams(dp(size), dp(size))
}

fun Context.row(drawableId: Int, title: String, detail: String, end: View? = null): LinearLayout = LinearLayout(this).apply {
    orientation = LinearLayout.HORIZONTAL
    gravity = Gravity.CENTER_VERTICAL
    minimumHeight = dp(64)
    setPadding(0, dp(8), 0, dp(8))
    addView(icon(drawableId, R.color.primary))
    addView(
        LinearLayout(context).apply {
            orientation = LinearLayout.VERTICAL
            addView(heading(title, 18f))
            addView(bodyText(detail, 14f).apply { isSingleLine = true })
        },
        LinearLayout.LayoutParams(0, LinearLayout.LayoutParams.WRAP_CONTENT, 1f).apply { marginStart = dp(20) },
    )
    end?.let { addView(it) }
}

fun Context.stepDots(step: Int, steps: Int): LinearLayout = LinearLayout(this).apply {
    orientation = LinearLayout.HORIZONTAL
    gravity = Gravity.CENTER_HORIZONTAL
    contentDescription = "Step $step of $steps"
    for (dot in 1..steps) {
        addView(
            View(context).apply {
                background =
                    GradientDrawable().apply {
                        cornerRadius = dp(3).toFloat()
                        setColor(colorOf(if (dot == step) R.color.primary else R.color.outline_variant))
                    }
            },
            LinearLayout.LayoutParams(dp(if (dot == step) 20 else 6), dp(6)).apply { marginEnd = dp(6) },
        )
    }
}

fun Context.statusBadge(drawableId: Int, backgroundId: Int, iconColorId: Int): View = LinearLayout(this).apply {
    gravity = Gravity.CENTER
    background = GradientDrawable().apply {
        shape = GradientDrawable.OVAL
        setColor(colorOf(backgroundId))
    }
    addView(icon(drawableId, iconColorId, 56))
    layoutParams = LinearLayout.LayoutParams(dp(120), dp(120))
}

fun LinearLayout.spacer() {
    addView(View(context), LinearLayout.LayoutParams(0, 0, 1f))
}

fun Activity.showScreen(content: View) {
    setContentView(
        FrameLayout(this).apply {
            fitsSystemWindows = true
            setBackgroundColor(colorOf(R.color.surface))
            addView(content, FrameLayout.LayoutParams(FrameLayout.LayoutParams.MATCH_PARENT, FrameLayout.LayoutParams.MATCH_PARENT))
        },
    )
}

fun Activity.firstRunColumn(step: Int, title: String): LinearLayout = column().apply {
    setPadding(dp(24), dp(32), dp(24), dp(24))
    addView(stepDots(step, FIRST_RUN_STEPS))
    addView(
        heading(title),
        LinearLayout.LayoutParams(LinearLayout.LayoutParams.WRAP_CONTENT, LinearLayout.LayoutParams.WRAP_CONTENT).apply {
            topMargin =
                dp(32)
        },
    )
}

const val FIRST_RUN_STEPS = 3

fun waitingIconOf(state: CompanionState.Waiting): Int = when (state) {
    CompanionState.WaitingForCharger -> R.drawable.ic_plug
    CompanionState.WaitingForUnmeteredNetwork -> R.drawable.ic_wifi
}
