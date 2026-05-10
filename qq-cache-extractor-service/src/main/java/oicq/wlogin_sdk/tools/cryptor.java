package oicq.wlogin_sdk.tools;

/* loaded from: classes7.dex */
public class cryptor {
    public static byte[] decrypt(byte[] bArr, int i, int i2, byte[] bArr2) {
        if (bArr != null && bArr2 != null) {
            byte[] bArr3 = new byte[i2];
            int i3 = 0;
            System.arraycopy(bArr, i, bArr3, 0, i2);
            byte[] bArr4 = new byte[bArr2.length];
            System.arraycopy(bArr2, 0, bArr4, 0, bArr2.length);
            a aVar = new a();
            aVar.e = 0;
            aVar.f1136d = 0;
            aVar.h = bArr4;
            int i4 = 8;
            byte[] bArr5 = new byte[8];
            if (i2 % 8 != 0 || i2 < 16) {
                return null;
            }
            byte[] a2 = aVar.a(bArr3, 0);
            aVar.b = a2;
            int i32 = a2[0] & 7;
            aVar.f = i32;
            int i42 = (i2 - i32) - 10;
            if (i42 < 0) {
                return null;
            }
            for (int i5 = 0; i5 < 8; i5++) {
                bArr5[i5] = 0;
            }
            aVar.c = new byte[i42];
            aVar.e = 0;
            aVar.f1136d = 8;
            aVar.j = 8;
            aVar.f++;
            aVar.g = 1;
            while (true) {
                int i6 = aVar.g;
                if (i6 <= 2) {
                    int i7 = aVar.f;
                    if (i7 < 8) {
                        aVar.f = i7 + 1;
                        aVar.g = i6 + 1;
                    }
                    if (aVar.f == 8) {
                        if (!aVar.a(bArr3, 0, i2)) {
                            return null;
                        }
                        bArr5 = bArr3;
                    }
                } else {
                    int i8 = 0;
                    while (i42 != 0) {
                        int i9 = aVar.f;
                        if (i9 < i4) {
                            aVar.c[i8] = (byte) (bArr5[(aVar.e + i3) + i9] ^ aVar.b[i9]);
                            i8++;
                            i42--;
                            aVar.f = i9 + 1;
                        }
                        if (aVar.f == 8) {
                            aVar.e = aVar.f1136d - 8;
                            if (!aVar.a(bArr3, 0, i2)) {
                                return null;
                            }
                            bArr5 = bArr3;
                        }
                        i3 = 0;
                        i4 = 8;
                    }
                    aVar.g = 1;
                    while (aVar.g < 8) {
                        int i10 = aVar.f;
                        if (i10 < 8) {
                            if ((bArr5[(aVar.e + 0) + i10] ^ aVar.b[i10]) != 0) {
                                return null;
                            }
                            aVar.f = i10 + 1;
                        }
                        if (aVar.f == 8) {
                            aVar.e = aVar.f1136d;
                            if (!aVar.a(bArr3, 0, i2)) {
                                return null;
                            }
                            bArr5 = bArr3;
                        }
                        aVar.g++;
                    }
                    return aVar.c;
                }
            }
        }
        return null;
    }

    public static byte[] encrypt(byte[] bArr, int i, int i2, byte[] bArr2) {
        int i3;
        int i22 = i2;
        if (bArr != null && bArr2 != null) {
            byte[] bArr3 = new byte[i22];
            byte b = 0;
            System.arraycopy(bArr, i, bArr3, 0, i22);
            byte[] bArr4 = new byte[bArr2.length];
            System.arraycopy(bArr2, 0, bArr4, 0, bArr2.length);
            a aVar = new a();
            int i4 = 8;
            byte[] bArr5 = new byte[8];
            aVar.f1135a = bArr5;
            aVar.b = new byte[8];
            aVar.f = 1;
            aVar.g = 0;
            aVar.e = 0;
            aVar.f1136d = 0;
            aVar.h = bArr4;
            aVar.i = true;
            int i42 = (i22 + 10) % 8;
            aVar.f = i42;
            if (i42 != 0) {
                aVar.f = 8 - i42;
            }
            aVar.c = new byte[aVar.f + i22 + 10];
            bArr5[0] = (byte) ((aVar.k.nextInt() & 248) | aVar.f);
            int i5 = 1;
            while (true) {
                i3 = aVar.f;
                if (i5 > i3) {
                    break;
                }
                aVar.f1135a[i5] = (byte) (aVar.k.nextInt() & 255);
                i5++;
                i4 = i4;
                b = 0;
            }
            aVar.f = i3 + 1;
            for (int i6 = 0; i6 < i4; i6++) {
                aVar.b[i6] = b;
            }
            aVar.g = 1;
            while (aVar.g <= 2) {
                int i7 = aVar.f;
                if (i7 < i4) {
                    byte[] bArr6 = aVar.f1135a;
                    aVar.f = i7 + 1;
                    bArr6[i7] = (byte) (aVar.k.nextInt() & 255);
                    aVar.g++;
                }
                if (aVar.f == i4) {
                    aVar.a();
                }
            }
            int i8 = 0;
            while (i22 > 0) {
                int i9 = aVar.f;
                if (i9 < i4) {
                    byte[] bArr7 = aVar.f1135a;
                    aVar.f = i9 + 1;
                    bArr7[i9] = bArr3[i8];
                    i22--;
                    i8++;
                }
                if (aVar.f == i4) {
                    aVar.a();
                }
            }
            aVar.g = 1;
            while (true) {
                int i10 = aVar.g;
                if (i10 <= 7) {
                    int i11 = aVar.f;
                    if (i11 < i4) {
                        byte[] bArr8 = aVar.f1135a;
                        aVar.f = i11 + 1;
                        bArr8[i11] = 0;
                        aVar.g = i10 + 1;
                    }
                    if (aVar.f == 8) {
                        aVar.a();
                    }
                    i4 = 8;
                } else {
                    return aVar.c;
                }
            }
        } else {
            return null;
        }
    }
}
