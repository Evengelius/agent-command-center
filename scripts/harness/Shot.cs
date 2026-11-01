using System;

// Shot captures a window.
public static class Shot {
    public static void Save(IntPtr h, string path) {
        var r = Term.Rect(h);
        int w = r.R - r.L, ht = r.B - r.T;
        using (var bmp = new System.Drawing.Bitmap(w, ht))
        using (var g = System.Drawing.Graphics.FromImage(bmp)) {
            g.CopyFromScreen(r.L, r.T, 0, 0, new System.Drawing.Size(w, ht));
            bmp.Save(path, System.Drawing.Imaging.ImageFormat.Png);
        }
    }
}
