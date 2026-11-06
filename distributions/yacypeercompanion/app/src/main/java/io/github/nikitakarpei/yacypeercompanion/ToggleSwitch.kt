package io.github.nikitakarpei.yacypeercompanion

import android.content.Context
import android.graphics.Canvas
import android.graphics.Paint
import android.graphics.RectF
import android.widget.CompoundButton
import android.widget.Switch

class ToggleSwitch(ctx: Context) : CompoundButton(ctx) {
    private val trackPaint = Paint(Paint.ANTI_ALIAS_FLAG)
    private val outlinePaint = Paint(Paint.ANTI_ALIAS_FLAG).apply {
        style = Paint.Style.STROKE
        strokeWidth = ctx.dp(2).toFloat()
    }
    private val thumbPaint = Paint(Paint.ANTI_ALIAS_FLAG)
    private val track = RectF()

    init {
        buttonDrawable = null
        background = null
        minWidth = 0
        minHeight = 0
        minimumWidth = 0
        minimumHeight = 0
        setPadding(0, 0, 0, 0)
    }

    override fun setChecked(checked: Boolean) {
        super.setChecked(checked)
        invalidate()
    }

    override fun onMeasure(widthMeasureSpec: Int, heightMeasureSpec: Int) {
        setMeasuredDimension(context.dp(52), context.dp(32))
    }

    override fun onDraw(canvas: Canvas) {
        val radius = height / 2f
        val inset = outlinePaint.strokeWidth / 2
        track.set(inset, inset, width - inset, height - inset)
        if (isChecked) {
            trackPaint.color = context.colorOf(R.color.primary)
            canvas.drawRoundRect(track, radius, radius, trackPaint)
            thumbPaint.color = context.colorOf(R.color.on_primary)
            canvas.drawCircle(width - radius, radius, context.dp(12).toFloat(), thumbPaint)
            return
        }
        trackPaint.color = context.colorOf(R.color.surface_container_highest)
        canvas.drawRoundRect(track, radius, radius, trackPaint)
        outlinePaint.color = context.colorOf(R.color.outline)
        canvas.drawRoundRect(track, radius, radius, outlinePaint)
        thumbPaint.color = context.colorOf(R.color.outline)
        canvas.drawCircle(radius, radius, context.dp(8).toFloat(), thumbPaint)
    }

    override fun getAccessibilityClassName(): CharSequence = Switch::class.java.name
}
