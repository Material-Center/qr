package oicq.wlogin_sdk.tools;

import java.io.ByteArrayOutputStream;
import java.io.DataOutputStream;
import java.io.IOException;
import java.util.Random;

/* loaded from: classes7.dex */
public class a {
    public byte[] b;
    public byte[] c;
    public int e;
    public int f;
    public byte[] f1135a;
    public int f1136d;
    public int g;
    public byte[] h;
    public int j;
    public boolean i = true;
    public Random k = new Random();

    public static long b(byte[] bArr, int i, int i2) {
        int i3;
        long j = 0;
        if (i2 > 4) {
            i3 = i + 4;
        } else {
            i3 = i2 + i;
        }
        while (i < i3) {
            j = (j << 8) | (bArr[i] & 255);
            i++;
        }
        return 4294967295L & j;
    }

    public final byte[] a(byte[] bArr, int i) {
        try {
            long b = b(bArr, i, 4);
            long b2 = b(bArr, i + 4, 4);
            long b3 = b(this.h, 0, 4);
            long b4 = b(this.h, 4, 4);
            long b5 = b(this.h, 8, 4);
            long b6 = b(this.h, 12, 4);
            int i2 = 16;
            long j = 3816266640L;
            while (true) {
                int i3 = i2 - 1;
                if (i2 <= 0) {
                    ByteArrayOutputStream byteArrayOutputStream = new ByteArrayOutputStream(8);
                    DataOutputStream dataOutputStream = new DataOutputStream(byteArrayOutputStream);
                    dataOutputStream.writeInt((int) b);
                    dataOutputStream.writeInt((int) b2);
                    dataOutputStream.close();
                    return byteArrayOutputStream.toByteArray();
                }
                b2 = (b2 - ((((b << 4) + b5) ^ (b + j)) ^ ((b >>> 5) + b6))) & 4294967295L;
                b = (b - ((((b2 << 4) + b3) ^ (b2 + j)) ^ ((b2 >>> 5) + b4))) & 4294967295L;
                j = (j - 2654435769L) & 4294967295L;
                i2 = i3;
            }
        } catch (IOException e) {
            return null;
        }
    }

    public final boolean a(byte[] bArr, int i, int i2) {
        this.f = 0;
        while (true) {
            int i3 = this.f;
            if (i3 >= 8) {
                byte[] a2 = a(this.b, 0);
                this.b = a2;
                if (a2 == null) {
                    return false;
                }
                this.j += 8;
                this.f1136d += 8;
                this.f = 0;
                return true;
            }
            if (this.j + i3 >= i2) {
                return true;
            }
            byte[] bArr2 = this.b;
            bArr2[i3] = (byte) (bArr2[i3] ^ bArr[(this.f1136d + i) + i3]);
            this.f = i3 + 1;
        }
    }

    public final void a() {
        int i;
        byte[] bArr;
        int i2 = 0;
        this.f = 0;
        while (true) {
            int i3 = this.f;
            i = 8;
            if (i3 >= 8) {
                break;
            }
            int i4 = i2;
            if (this.i) {
                byte[] bArr2 = this.f1135a;
                bArr2[i3] = (byte) (bArr2[i3] ^ this.b[i3]);
            } else {
                byte[] bArr3 = this.f1135a;
                bArr3[i3] = (byte) (bArr3[i3] ^ this.c[this.e + i3]);
            }
            this.f = i3 + 1;
            i2 = i4;
        }
        byte[] bArr4 = this.f1135a;
        try {
            long b = b(bArr4, i2, 4);
            long b2 = b(bArr4, 4, 4);
            long b3 = b(this.h, i2, 4);
            long b4 = b(this.h, 4, 4);
            long b5 = b(this.h, 8, 4);
            long b6 = b(this.h, 12, 4);
            int i22 = 16;
            long j = 0;
            while (true) {
                int i32 = i22 - 1;
                if (i22 <= 0) {
                    break;
                }
                j = (j + 2654435769L) & 4294967295L;
                b = (b + ((((b2 << 4) + b3) ^ (b2 + j)) ^ ((b2 >>> 5) + b4))) & 4294967295L;
                b2 = (b2 + ((((b << 4) + b5) ^ (b + j)) ^ ((b >>> 5) + b6))) & 4294967295L;
                i22 = i32;
                i = 8;
            }
            ByteArrayOutputStream byteArrayOutputStream = new ByteArrayOutputStream(i);
            DataOutputStream dataOutputStream = new DataOutputStream(byteArrayOutputStream);
            dataOutputStream.writeInt((int) b);
            dataOutputStream.writeInt((int) b2);
            dataOutputStream.close();
            byte[] bArr5 = byteArrayOutputStream.toByteArray();
            bArr = bArr5;
        } catch (IOException e) {
            bArr = null;
        }
        System.arraycopy(bArr, 0, this.c, this.f1136d, 8);
        this.f = 0;
        while (true) {
            int i42 = this.f;
            if (i42 >= 8) {
                System.arraycopy(this.f1135a, 0, this.b, 0, 8);
                int i5 = this.f1136d;
                this.e = i5;
                this.f1136d = i5 + 8;
                this.f = 0;
                this.i = false;
                return;
            }
            byte[] bArr52 = this.c;
            int i6 = this.f1136d + i42;
            bArr52[i6] = (byte) (bArr52[i6] ^ this.b[i42]);
            this.f = i42 + 1;
        }
    }
}
