package io.github.nikitakarpei.yacypeercompanion

import android.content.Context
import android.graphics.Canvas
import android.graphics.Paint
import android.graphics.Path
import android.graphics.RectF
import android.view.View

class ScanFrame(ctx: Context) : View(ctx) {
    private val stroke = ctx.dp(4).toFloat()
    private val radius = ctx.dp(20).toFloat()
    private val arm = ctx.dp(44).toFloat()
    private val paint =
        Paint(Paint.ANTI_ALIAS_FLAG).apply {
            style = Paint.Style.STROKE
            strokeWidth = stroke
            strokeCap = Paint.Cap.ROUND
            color = 0xFFFFFFFF.toInt()
        }

    private val corner =
        Path().apply {
            val inset = stroke / 2
            moveTo(inset, arm)
            lineTo(inset, radius)
            arcTo(RectF(inset, inset, inset + 2 * radius, inset + 2 * radius), 180f, 90f)
            lineTo(arm, inset)
        }

    override fun onDraw(canvas: Canvas) {
        super.onDraw(canvas)
        for (turn in 0 until 4) {
            canvas.save()
            canvas.rotate(90f * turn, width / 2f, height / 2f)
            canvas.drawPath(corner, paint)
            canvas.restore()
        }
    }
}
