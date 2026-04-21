package oicq.wlogin_sdk.tools;

import android.content.Context;
import android.content.pm.ApplicationInfo;
import android.net.Proxy;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.os.Environment;
import android.os.Process;
import android.util.Log;
import androidx.core.view.ViewCompat;
import com.google.common.base.Ascii;
import java.io.ByteArrayInputStream;
import java.io.ByteArrayOutputStream;
import java.io.File;
import java.io.FileInputStream;
import java.io.FilenameFilter;
import java.io.IOException;
import java.io.PrintWriter;
import java.io.StringWriter;
import java.io.Writer;
import java.security.Key;
import java.security.KeyFactory;
import java.security.KeyPair;
import java.security.KeyPairGenerator;
import java.security.SecureRandom;
import java.security.spec.PKCS8EncodedKeySpec;
import java.security.spec.X509EncodedKeySpec;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.HashMap;
import java.util.Random;
import java.util.zip.DeflaterOutputStream;
import java.util.zip.InflaterInputStream;
import javax.crypto.Cipher;
import oicq.wlogin_sdk.request.u;
import okhttp3.HttpUrl;
import org.bouncycastle.pqc.math.linearalgebra.Matrix;

/* loaded from: classes7.dex */
public class util {
    public static final int ASYNC_GET_SALT_UIN_LIST = 24;
    public static final int ASYNC_GET_ST_BY_GATEWAY = 21;
    public static final int ASYNC_GET_ST_BY_PHONE = 22;
    public static final int ASYNC_GET_ST_BY_PHONE_PASSWORD = 23;
    public static final int ASYNC_GET_ST_BY_THIRD_PLATFORM = 25;
    public static final int ASYN_CHECK_IMAGE = 2;
    public static final int ASYN_CHECK_SMS = 4;
    public static final int ASYN_EXCEPTION = 11;
    public static final int ASYN_GET_A1_WITH_A1 = 6;
    public static final int ASYN_GET_ST_WITHOUT_PWD = 5;
    public static final int ASYN_GET_ST_WITH_PWD = 0;
    public static final int ASYN_QUICKLOG_BY_GATEWAY = 19;
    public static final int ASYN_QUICKLOG_BY_THIRD_PLATFORM = 20;
    public static final int ASYN_QUICKLOG_BY_WECHAT = 18;
    public static final int ASYN_QUICKLOG_WITH_PTSIG = 16;
    public static final int ASYN_QUICKLOG_WITH_QQSIG = 15;
    public static final int ASYN_QUICKLOG_WITH_QRSIG = 17;
    public static final int ASYN_REFLUSH_IMAGE = 1;
    public static final int ASYN_REFLUSH_SMS = 3;
    public static final int ASYN_REPORT = 7;
    public static final int ASYN_REPORT_ERROR = 8;
    public static final int ASYN_SMSLOGIN_CHECK = 12;
    public static final int ASYN_SMSLOGIN_REFRESH = 14;
    public static final int ASYN_SMSLOGIN_VERIFY = 13;
    public static final int ASYN_TRANSPORT = 9;
    public static final int ASYN_TRANSPORT_MSF = 10;
    public static final long BUILD_TIME = 1697015435;
    public static final int BUSINESS_TYPE_LOGIN_GATEWAY = 2;
    public static final int BUSINESS_TYPE_LOGIN_SMS = 3;
    public static final int BUSINESS_TYPE_NULL = 0;
    public static final String CMD_DEVICE_LOCK = "wtlogin.device_lock";
    public static final String CMD_LOG_REPORT = "wtlogin.log_report";
    public static final String CMD_QR_LOGIN = "wtlogin.qrlogin";
    public static final String CMD_REGISTER = "wtlogin.register";
    public static final int D = 2;
    private static SimpleDateFormat DAYFORMAT = null;
    public static final int E_A1_DECRYPT = -1014;
    public static final int E_A1_FORMAT = -1016;
    public static final int E_A1_SEQ_ERROR = 20;
    public static final int E_ADVANCE_NOTICE = 257;
    public static final int E_APK_CHK_ERR = -1021;
    public static final int E_BUFFER_OVERFLOW = -1023;
    public static final int E_DECRYPT = -1002;
    public static final int E_ENCODING = -1013;
    public static final int E_ENCRYPTION_METHOD = -1024;
    public static final int E_GATEWAY_LOGIN_FAILED = -2005;
    public static final int E_GATEWAY_LOGIN_NUM_FAILED = -2004;
    public static final int E_INPUT = -1017;
    public static final int E_LOGIN_THROUGH_QQ = -2001;
    public static final int E_LOGIN_THROUGH_WEB = -2000;
    public static final int E_NAME_INVALID = -1008;
    public static final int E_NEWST_DECRYPT = -1025;
    public static final int E_NO_KEY = -1004;
    public static final int E_NO_NETWORK = -1026;
    public static final int E_NO_REG_CMD = -1010;
    public static final int E_NO_RET = -1000;
    public static final int E_NO_TGTKEY = -1006;
    public static final int E_NO_UIN = -1003;
    public static final int E_OTHER_EXCEPTION = -2006;
    public static final int E_PENDING = -1001;
    public static final int E_PK_LEN = -1009;
    public static final int E_PUSH_REG = -1011;
    public static final int E_RESOLVE_ADDR = -1007;
    public static final int E_SAVE_TICKET_ERROR = -1022;
    public static final int E_SHARE_SERVICE_EXCEPTION = -1020;
    public static final int E_SHARE_SERVICE_PARAM = -1019;
    public static final int E_SHARE_SERVICE_UNCHECK = -1018;
    public static final int E_SYSTEM = -1012;
    public static final int E_TLV_DECRYPT = -1015;
    public static final int E_TLV_VERIFY = -1005;
    public static final int E_WXLOGIN_NO_REGISTER = 230;
    public static final int E_WXLOGIN_NUM_FAILED = -2003;
    public static final int E_WXLOGIN_TOKEN_FAILED = -2002;
    public static final String FILE_DIR = "wtlogin";
    public static int GUID_DELAY_HOUR = 0;
    private static int HONEYCOMB = 0;
    public static final String HTTPS_PREFIX = "https://";
    public static final String HTTPS_WLOGIN_PATH = "/cgi-bin/wlogin_proxy_login";
    public static final int I = 1;
    public static final int KEY_TLV543_IN_TLV199 = 1676611;
    public static boolean LOGCAT_OUT = false;
    public static final String LOG_DIR = "tencent/wtlogin";
    public static int LOG_LEVEL = 0;
    public static String LOG_TAG_EVENT_REPORT = null;
    public static String LOG_TAG_GATEWAY_LOGIN_NEW_DOV = null;
    public static String LOG_TAG_POW = null;
    public static String LOG_TAG_PRIVACY = null;
    public static String LOG_TAG_QIMEI = null;
    public static int MAX_APPID = 0;
    public static final int MAX_CONTENT_SIZE = 40960;
    public static final int MAX_FILE_SIZE = 524288;
    public static final int MAX_INIT_KEY_TIME = 3;
    public static int MAX_NAME_LEN = 0;
    public static final int MAX_REQUEST_COUNT_OF_PSKEY = 20;
    private static int MODE_MULTI_PROCESS = 0;
    public static final int QQ_APP_ID = 16;
    public static final String RANDOM_ANDROID_ID = "random_AndroidId";
    public static final int REG_CHECK_ERROR_FACE = 59;
    public static final int REG_OVER_LIMIT = 61;
    public static final String SDK_VERSION = "6.0.0.2556";
    public static final int SSO_VERSION = 21;
    public static final long SVN_VER = 2556;
    public static final int S_BABYLH_EXPIRED = 116;
    public static final int S_GET_IMAGE = 2;
    public static final int S_GET_SMS = 160;
    public static final int S_GET_SMS_TOKEN = 239;
    public static final int S_LH_EXPIRED = 41;
    public static final int S_PHONE_DEV = 224;
    public static final int S_PWD_WRONG = 1;
    public static final int S_ROLL_BACK = 180;
    public static final int S_SEC_GUID = 204;
    public static final int S_SUCCESS = 0;
    public static final String TAG = "wlogin_sdk";
    public static final int TLV542 = 1346;
    public static final int W = 0;
    public static final String WT_LOGIN_URL_HOST = "txz.qq.com";
    public static final char[] base64_encode_chars;
    public static final char base64_pad_url = '_';
    public static final short[] base64_reverse_table_url;
    private static boolean libwtecdh_loaded;
    public static boolean loadEncryptSo;
    public static String logContent;
    public static HashMap<Long, String> roleCmdMap;

    static {
        DAYFORMAT = null;
        GUID_DELAY_HOUR = 0;
        HONEYCOMB = 0;
        LOGCAT_OUT = false;
        LOG_LEVEL = 0;
        LOG_TAG_EVENT_REPORT = null;
        LOG_TAG_GATEWAY_LOGIN_NEW_DOV = null;
        LOG_TAG_POW = null;
        LOG_TAG_PRIVACY = null;
        LOG_TAG_QIMEI = null;
        MAX_APPID = 0;
        MAX_NAME_LEN = 0;
        MODE_MULTI_PROCESS = 0;
        HashMap<Long, String> hashMap = new HashMap<>();
        roleCmdMap = hashMap;
        hashMap.put(85L, CMD_LOG_REPORT);
        roleCmdMap.put(95L, CMD_REGISTER);
        roleCmdMap.put(505L, CMD_DEVICE_LOCK);
        roleCmdMap.put(114L, CMD_QR_LOGIN);
        MAX_APPID = 65535;
        MAX_NAME_LEN = 128;
        LOG_LEVEL = 1;
        LOGCAT_OUT = false;
        LOG_TAG_GATEWAY_LOGIN_NEW_DOV = "gateway_login_new_dov";
        LOG_TAG_POW = "pow";
        LOG_TAG_EVENT_REPORT = "event_report";
        LOG_TAG_PRIVACY = "privacy";
        LOG_TAG_QIMEI = "qimei";
        GUID_DELAY_HOUR = 360;
        logContent = "";
        DAYFORMAT = null;
        libwtecdh_loaded = false;
        loadEncryptSo = true;
        MODE_MULTI_PROCESS = 4;
        HONEYCOMB = 11;
        base64_encode_chars = new char[]{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', Matrix.MATRIX_TYPE_RANDOM_LT, 'M', 'N', 'O', 'P', 'Q', Matrix.MATRIX_TYPE_RANDOM_REGULAR, 'S', 'T', Matrix.MATRIX_TYPE_RANDOM_UT, 'V', 'W', 'X', 'Y', Matrix.MATRIX_TYPE_ZERO, 'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p', 'q', 'r', 's', 't', 'u', 'v', 'w', 'x', 'y', 'z', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '+', '/'};
        base64_reverse_table_url = new short[]{-1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, 62, -1, -1, 63, -1, -1, 52, 53, 54, 55, 56, 57, 58, 59, 60, 61, -1, -1, -1, -1, -1, -1, -1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, -1, -1, -1, -1, -1, -1, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1, -1};
    }

    /* loaded from: classes7.dex */
    public static class a implements FilenameFilter {
        @Override // java.io.FilenameFilter
        public boolean accept(File file, String str) {
            return str.matches("wtlogin_[0-9]+\\.log");
        }
    }

    public static boolean ExistSDCard() {
        return Environment.getExternalStorageState().equals("mounted");
    }

    public static void LOGD(String str) {
        try {
            Log.d(TAG, str);
        } catch (Exception e) {
        }
    }

    public static void LOGD(String str, String str2) {
        try {
            Log.d(TAG, str);
        } catch (Exception e) {
        }
    }

    public static void LOGI(String str) {
        try {
            Log.d(TAG, str);
        } catch (Exception e) {
        }
    }

    public static void LOGI(String str, String str2) {
        try {
            Log.d(TAG, str);
        } catch (Exception e) {
        }
    }

    public static void LOGW(String str, String str2) {
        try {
            Log.d(TAG, str);
        } catch (Exception e) {
        }
    }

    public static void LOGW(String str, String str2, String str3) {
        try {
            Log.d(TAG, str);
        } catch (Exception e) {
        }
    }

    public static byte[] RSADecrypt(byte[] bArr, Key key) {
        String str;
        if (bArr == null || key == null) {
            str = "data or key is null";
        } else {
            try {
                Cipher cipher = Cipher.getInstance("RSA/ECB/PKCS1Padding");
                cipher.init(2, key);
                int length = bArr.length;
                byte[] bArr2 = new byte[length];
                if (length % 128 != 0) {
                    LOGI("len not match block size", "");
                    return null;
                }
                int i = 0;
                for (int i2 = 0; i2 < length / 128; i2++) {
                    byte[] bArr3 = new byte[128];
                    System.arraycopy(bArr, i2 * 128, bArr3, 0, 128);
                    byte[] doFinal = cipher.doFinal(bArr3);
                    System.arraycopy(doFinal, 0, bArr2, i, doFinal.length);
                    i += doFinal.length;
                }
                byte[] bArr4 = new byte[i];
                System.arraycopy(bArr2, 0, bArr4, 0, i);
                return bArr4;
            } catch (Exception e) {
                str = "descypt exception:" + e.toString();
            }
        }
        LOGI(str, "");
        return null;
    }

    public static byte[] RSAEncrypt(byte[] bArr, Key key) {
        if (bArr != null && key != null) {
            try {
                Cipher cipher = Cipher.getInstance("RSA/ECB/PKCS1Padding");
                cipher.init(1, key);
                int length = bArr.length;
                int round = (int) Math.round((length / 117) + 0.5d);
                byte[] bArr2 = new byte[round * 128];
                int i = 117;
                for (int i2 = 0; i2 < round; i2++) {
                    if (length < 117) {
                        i = length;
                    }
                    byte[] bArr3 = new byte[i];
                    System.arraycopy(bArr, i2 * 117, bArr3, 0, i);
                    System.arraycopy(cipher.doFinal(bArr3), 0, bArr2, i2 * 128, 128);
                    length -= i;
                }
                return bArr2;
            } catch (Exception e) {
                return null;
            }
        }
        return null;
    }

    public static Key RSAPrivKeyFromJNI(byte[] bArr) {
        if (bArr == null) {
            return null;
        }
        byte[] bArr2 = {48, -126, 2, 117, 2, 1, 0, 48, Ascii.CR, 6, 9, 42, -122, 72, -122, -9, Ascii.CR, 1, 1, 1, 5, 0, 4, -126, 2, 95};
        int length = bArr.length - 607;
        bArr2[3] = (byte) (bArr2[3] + length);
        bArr2[25] = (byte) (bArr2[25] + length);
        byte[] bArr3 = new byte[bArr.length + 26];
        System.arraycopy(bArr2, 0, bArr3, 0, 26);
        System.arraycopy(bArr, 0, bArr3, 26, bArr.length);
        try {
            return KeyFactory.getInstance("RSA").generatePrivate(new PKCS8EncodedKeySpec(bArr3));
        } catch (Exception e) {
            printException(e, "");
            return null;
        }
    }

    public static byte[] RSAPrivKeyFromJava(byte[] bArr) {
        if (bArr == null) {
            return null;
        }
        try {
            byte[] encoded = KeyFactory.getInstance("RSA").generatePrivate(new PKCS8EncodedKeySpec(bArr)).getEncoded();
            int length = encoded.length - 26;
            byte[] bArr2 = new byte[length];
            System.arraycopy(encoded, 26, bArr2, 0, length);
            return bArr2;
        } catch (Exception e) {
            printException(e, "");
            return null;
        }
    }

    public static Key RSAPubKeyFromJNI(byte[] bArr) {
        if (bArr == null) {
            return null;
        }
        byte[] bArr2 = new byte[bArr.length + 22];
        System.arraycopy(new byte[]{48, -127, -97, 48, Ascii.CR, 6, 9, 42, -122, 72, -122, -9, Ascii.CR, 1, 1, 1, 5, 0, 3, -127, -115, 0}, 0, bArr2, 0, 22);
        System.arraycopy(bArr, 0, bArr2, 22, bArr.length);
        try {
            return KeyFactory.getInstance("RSA").generatePublic(new X509EncodedKeySpec(bArr2));
        } catch (Exception e) {
            printException(e, "");
            return null;
        }
    }

    public static byte[] RSAPubKeyFromJava(byte[] bArr) {
        if (bArr == null) {
            return null;
        }
        try {
            byte[] encoded = KeyFactory.getInstance("RSA").generatePublic(new X509EncodedKeySpec(bArr)).getEncoded();
            int length = encoded.length - 22;
            byte[] bArr2 = new byte[length];
            System.arraycopy(encoded, 22, bArr2, 0, length);
            return bArr2;
        } catch (Exception e) {
            printException(e, "");
            return null;
        }
    }

    public static byte[] base64_decode_url(byte[] r11, int r12) {
        throw new UnsupportedOperationException("Method not decompiled: oicq.wlogin_sdk.tools.util.base64_decode_url(byte[], int):byte[]");
    }

    public static String base64_encode(byte[] bArr) {
        char c = 0;
        StringBuffer stringBuffer = new StringBuffer();
        int length = bArr.length;
        if (0 < length) {
            int i2 = 0 + 1;
            int i3 = bArr[0] & 255;
            if (i2 == length) {
                char[] cArr = base64_encode_chars;
                stringBuffer.append(cArr[i3 >>> 2]);
                c = cArr[(i3 & 3) << 4];
            } else {
                int i4 = i2 + 1;
                int i5 = bArr[i2] & 255;
                if (i4 == length) {
                    char[] cArr2 = base64_encode_chars;
                    stringBuffer.append(cArr2[i3 >>> 2]);
                    stringBuffer.append(cArr2[((i3 & 3) << 4) | ((i5 & 240) >>> 4)]);
                    c = cArr2[(i5 & 15) << 2];
                } else {
                    int i = i4 + 1;
                    int i7 = bArr[i4] & 255;
                    char[] cArr3 = base64_encode_chars;
                    stringBuffer.append(cArr3[i3 >>> 2]);
                    stringBuffer.append(cArr3[((i3 & 3) << 4) | ((i5 & 240) >>> 4)]);
                    stringBuffer.append(cArr3[((i5 & 15) << 2) | ((i7 & 192) >>> 6)]);
                    stringBuffer.append(cArr3[i7 & 63]);
                }
            }
            stringBuffer.append(c);
        }
        return stringBuffer.toString();
    }

    public static long buf_len(byte[] bArr) {
        if (bArr == null) {
            return 0L;
        }
        return bArr.length;
    }

    public static int buf_to_int16(byte[] bArr, int i) {
        return ((bArr[i] << 8) & 65280) + ((bArr[i + 1] << 0) & 255);
    }

    public static int buf_to_int32(byte[] bArr, int i) {
        return ((bArr[i] << Ascii.CAN) & ViewCompat.MEASURED_STATE_MASK) + ((bArr[i + 1] << 16) & 16711680) + ((bArr[i + 2] << 8) & 65280) + ((bArr[i + 3] << 0) & 255);
    }

    public static long buf_to_int64(byte[] bArr, int i) {
        return ((bArr[i] << 56) & (-72057594037927936L)) + 0 + ((bArr[i + 1] << 48) & 71776119061217280L) + ((bArr[i + 2] << 40) & 280375465082880L) + ((bArr[i + 3] << 32) & 1095216660480L) + ((bArr[i + 4] << Ascii.CAN) & 4278190080L) + ((bArr[i + 5] << 16) & 16711680) + ((bArr[i + 6] << 8) & 65280) + ((bArr[i + 7] << 0) & 255);
    }

    public static int buf_to_int8(byte[] bArr, int i) {
        return bArr[i] & 255;
    }

    public static String buf_to_string(byte[] bArr) {
        String str = "";
        if (bArr == null) {
            return "";
        }
        for (int i = 0; i < bArr.length; i++) {
            str = str + Integer.toHexString((bArr[i] >> 4) & 15) + Integer.toHexString(bArr[i] & Ascii.SI);
        }
        return str;
    }

    public static String buf_to_string(byte[] bArr, int i) {
        String str = "";
        if (bArr == null) {
            return "";
        }
        if (i > bArr.length) {
            i = bArr.length;
        }
        for (int i2 = 0; i2 < i; i2++) {
            str = str + Integer.toHexString((bArr[i2] >> 4) & 15) + Integer.toHexString(bArr[i2] & Ascii.SI);
        }
        return str;
    }

    public static Boolean check_uin_account(String str) {
        Boolean bool = Boolean.FALSE;
        LOGI("check_uin_account account = " + str, "");
        try {
            long parseLong = Long.parseLong(str);
            if (parseLong >= 10000 && parseLong <= 4294967295L) {
                return Boolean.TRUE;
            }
        } catch (NumberFormatException e) {
        }
        return bool;
    }

    public static void chg_retry_type(Context context) {
        set_net_retry_type(context, get_net_retry_type(context) == 0 ? 1 : 0);
    }

    public static byte[] compress(byte[] bArr) {
        if (bArr == null || bArr.length == 0) {
            return bArr;
        }
        ByteArrayOutputStream byteArrayOutputStream = new ByteArrayOutputStream();
        try {
            DeflaterOutputStream deflaterOutputStream = new DeflaterOutputStream(byteArrayOutputStream);
            deflaterOutputStream.write(bArr);
            deflaterOutputStream.close();
            return byteArrayOutputStream.toByteArray();
        } catch (IOException e) {
            return new byte[0];
        }
    }

    public static long constructSalt() {
        return (get_rand_32() << 32) + get_rand_32();
    }

    public static void decompress(byte[] bArr) {
        if (bArr == null || bArr.length == 0) {
            return;
        }
        LOGI("data len:" + bArr.length);
        int i = 0;
        int i2 = 0;
        while (bArr.length > i + 3) {
            int buf_to_int32 = buf_to_int32(bArr, i);
            if (bArr.length <= i + buf_to_int32 + 3) {
                return;
            }
            byte[] bArr2 = new byte[buf_to_int32];
            int i3 = i + 4;
            System.arraycopy(bArr, i3, bArr2, 0, buf_to_int32);
            i = i3 + buf_to_int32;
            i2++;
            LOGI("len:" + buf_to_int32);
            ByteArrayOutputStream byteArrayOutputStream = new ByteArrayOutputStream();
            try {
                InflaterInputStream inflaterInputStream = new InflaterInputStream(new ByteArrayInputStream(bArr2));
                byte[] bArr3 = new byte[1024];
                while (true) {
                    int read = inflaterInputStream.read(bArr3);
                    if (read == -1) {
                        break;
                    } else {
                        byteArrayOutputStream.write(bArr3, 0, read);
                    }
                }
                LOGI(i2 + byteArrayOutputStream.toString());
            } catch (IOException e) {
            }
        }
    }

    public static void deleteExpireFile(String str, int i) {
        File[] listFiles;
        if (str == null || str.length() == 0) {
            return;
        }
        LOGI("file path:" + str);
        try {
            File file = new File(str);
            if (file.isDirectory() && (listFiles = file.listFiles(new a())) != null) {
                int length = listFiles.length;
                for (int i2 = 0; i2 < length; i2++) {
                    if (!listFiles[i2].isDirectory() && (System.currentTimeMillis() - listFiles[i2].lastModified()) / 1000 > i) {
                        listFiles[i2].delete();
                    }
                }
            }
        } catch (Exception e) {
        }
    }

    public static void deleteExpireLog(Context context) {
        String str;
        if (context == null) {
            return;
        }
        try {
            String str2 = u.r0;
            if (str2 != null && str2.length() != 0) {
                str = u.r0;
            } else if (!ExistSDCard()) {
                deleteExpireFile(context.getFilesDir().getPath() + "/" + LOG_DIR, 259200);
                return;
            } else {
                File externalCacheDir = context.getExternalCacheDir();
                str = externalCacheDir.getAbsolutePath() + "/" + LOG_DIR + "/" + context.getPackageName();
            }
            deleteExpireFile(str, 691200);
        } catch (Exception e) {
        }
    }

    public static int format_ret_code(int i) {
        if (i == -1015 || i == -1014 || i == -1006 || i == -1002) {
            return 5;
        }
        if (i != -1000) {
            if (i != 0) {
                return i != 2 ? 17 : 2;
            }
            return 0;
        }
        return 1;
    }

    public static KeyPair generateRSAKeyPair() {
        try {
            KeyPairGenerator keyPairGenerator = KeyPairGenerator.getInstance("RSA");
            keyPairGenerator.initialize(1024);
            return keyPairGenerator.generateKeyPair();
        } catch (Exception e) {
            return null;
        }
    }

    public static byte[] getAppName(Context context) {
        String charSequence;
        try {
            ApplicationInfo applicationInfo = context.getPackageManager().getApplicationInfo(context.getPackageName(), 0);
            if (applicationInfo != null && (charSequence = context.getPackageManager().getApplicationLabel(applicationInfo).toString()) != null) {
                return charSequence.getBytes();
            }
        } catch (Throwable th) {
        }
        return new byte[0];
    }

    public static String getBaseband() {
        try {
            Class<?> cls = Class.forName("android.os.SystemProperties");
            return (String) cls.getMethod("get", String.class, String.class).invoke(cls.newInstance(), "gsm.version.baseband", "no message");
        } catch (Exception e) {
            return "";
        }
    }

    public static String getBootId() {
        throw new UnsupportedOperationException("Method not decompiled: oicq.wlogin_sdk.tools.util.getBootId():java.lang.String");
    }

    public static int getByteLength(byte[] bArr) {
        if (bArr == null) {
            return 0;
        }
        return bArr.length;
    }

    public static String getCurrentDay() {
        try {
            if (DAYFORMAT == null) {
                DAYFORMAT = new SimpleDateFormat("yyyyMMdd");
            }
            return DAYFORMAT.format(new Date());
        } catch (Exception e) {
            return null;
        }
    }

    public static String getCurrentPid() {
        return "[" + Process.myPid() + "]";
    }

    public static String getDate() {
        try {
            return "[" + System.currentTimeMillis() + "]";
        } catch (Exception e) {
            return "";
        }
    }

    public static long getFileModifyTime(String str) {
        File file = null;
        if (str == null) {
            return 0L;
        }
        if (str.length() != 0) {
            try {
                File file2 = new File(str);
                if (file2.exists()) {
                    file2.isFile();
                }
                return 0L;
            } catch (Exception e) {
                return 0L;
            }
        }
        return file.lastModified();
    }

    public static int getFileSize(String str) {
        try {
            File file = new File(str);
            if (!file.exists() || !file.isFile()) {
                return 0;
            }
            return (int) file.length();
        } catch (Exception e) {
            return 0;
        }
    }

    public static String getInnerVersion() {
        String str = Build.DISPLAY;
        return str.contains(Build.VERSION.INCREMENTAL) ? str : Build.VERSION.INCREMENTAL;
    }

    public static String getLanguage(Context context) {
        String country = context.getResources().getConfiguration().locale.getCountry();
        if (country.equals("CN")) {
            return "CN";
        }
        return country.equals("TW") ? "TW" : "EN";
    }

    public static String getLogDir(Context context) {
        String str = u.r0;
        if (str == null || str.length() == 0) {
            try {
                if (!ExistSDCard()) {
                    String path = context.getFilesDir().getPath();
                    return path + "/" + LOG_DIR;
                }
                return context.getExternalCacheDir().getAbsolutePath() + "/" + LOG_DIR + "/" + context.getPackageName();
            } catch (Exception e) {
                return "";
            }
        }
        return u.r0;
    }

    public static String getLogFileName(Context context, String str) {
        if (context == null || str == null || str.length() == 0) {
            return null;
        }
        String logDir = getLogDir(context);
        return logDir + "/wtlogin_" + str + ".log";
    }

    public static long getLogModifyTime(Context context, String str) {
        if (context == null || str == null || str.length() == 0) {
            return 0L;
        }
        return getFileModifyTime(getLogFileName(context, str));
    }

    public static String getMaskBytes(byte[] bArr, int i, int i2) {
        if (bArr == null) {
            return "null";
        }
        String str = new String(bArr);
        return i + i2 > str.length() ? "***" : str.substring(0, i) + "***" + str.substring(str.length() - i2);
    }

    public static String getMaskString(String str, int i, int i2) {
        return i + i2 > str.length() ? "***" : str.substring(0, i) + "***" + str.substring(str.length() - i2);
    }

    public static byte[] getRequestInitTime() {
        byte[] bArr = new byte[4];
        int64_to_buf32(bArr, 0, (System.currentTimeMillis() / 1000) + u.c0);
        return bArr;
    }

    public static String getSvnVersion() {
        return "[2556]";
    }

    public static String getThreadId() {
        return "[" + Thread.currentThread().getId() + "]";
    }

    public static String getThrowableInfo(Throwable th) {
        StringWriter stringWriter = new StringWriter();
        PrintWriter printWriter = new PrintWriter((Writer) stringWriter, true);
        th.printStackTrace(printWriter);
        printWriter.flush();
        stringWriter.flush();
        return stringWriter.toString();
    }

    public static String getUser(String str) {
        if (str != null) {
            return "[" + str + "]";
        }
        return HttpUrl.PATH_SEGMENT_ENCODE_SET_URI;
    }

    public static byte[] get_apk_id(Context context) {
        try {
            return context.getPackageName().getBytes();
        } catch (Throwable th) {
            return new byte[0];
        }
    }

    public static String get_apn_string(Context context) {
        return "wifi";
    }

    public static byte get_char(byte b) {
        int i;
        if (b < 48 || b > 57) {
            byte b2 = 97;
            if (b < 97 || b > 102) {
                b2 = 65;
                if (b < 65 || b > 70) {
                    return (byte) 0;
                }
            }
            i = (b - b2) + 10;
        } else {
            i = b - 48;
        }
        return (byte) i;
    }

    public static String get_mpasswd() {
        try {
            String str = "";
            for (byte b : SecureRandom.getSeed(16)) {
                int abs = Math.abs(b % Ascii.SUB) + (new Random().nextBoolean() ? 97 : 65);
                str = str + String.valueOf((char) abs);
            }
            return str;
        } catch (Throwable th) {
            return "1234567890123456";
        }
    }

    public static int get_net_retry_type(Context context) {
        return 0;
    }

    public static byte[] get_os_type() {
        return "android".getBytes();
    }

    public static byte[] get_os_version() {
        return Build.VERSION.RELEASE.getBytes();
    }

    public static String get_proxy_ip() {
        return Build.VERSION.SDK_INT < HONEYCOMB ? Proxy.getDefaultHost() : System.getProperty("http.proxyHost");
    }

    public static int get_proxy_port() {
        if (Build.VERSION.SDK_INT < HONEYCOMB) {
            return Proxy.getDefaultPort();
        }
        try {
            return Integer.parseInt(System.getProperty("http.proxyPort"));
        } catch (NumberFormatException e) {
            e.printStackTrace();
            return -1;
        }
    }

    public static byte[] get_rand_16byte(byte[] bArr) {
        return new byte[0];
    }

    public static int get_rand_32() {
        return (int) (Math.random() * 2.147483647E9d);
    }

    public static String get_release_time() {
        return "2023/10/11 17:10:35";
    }

    public static byte[] get_rsa_privkey(Context context) {
        byte[] bArr = new byte[0];
        return bArr.length <= 0 ? new byte[0] : bArr;
    }

    public static byte[] get_rsa_pubkey(Context context) {
        byte[] bArr = new byte[0];
        return bArr.length <= 0 ? new byte[0] : bArr;
    }

    public static byte[] get_saved_android_id(Context context) {
        byte[] bArr = new byte[0];
        return bArr;
    }

    public static int get_saved_network_type(Context context) {
        return 0;
    }

    public static long get_server_cur_time() {
        SecureRandom secureRandom = u.t;
        return (System.currentTimeMillis() / 1000) + u.c0;
    }

    public static byte[] get_server_host1(Context context) {
        return new byte[0];
    }

    public static byte[] get_server_host2(Context context) {
        return new byte[0];
    }

    public static byte[] get_server_ipv6_host1(Context context) {
        return new byte[0];
    }

    public static byte[] get_server_ipv6_host2(Context context) {
        return new byte[0];
    }

    public static byte[] get_ssid_addr(Context context) {
        return new byte[0];
    }

    public static byte[] get_wap_server_host1(Context context) {
        return new byte[0];
    }

    public static byte[] get_wap_server_host2(Context context) {
        return new byte[0];
    }

    public static byte[] get_wap_server_ipv6_host1(Context context) {
        return new byte[0];
    }

    public static byte[] get_wap_server_ipv6_host2(Context context) {
        return new byte[0];
    }

    public static void int16_to_buf(byte[] bArr, int i, int i2) {
        bArr[i + 1] = (byte) (i2 >> 0);
        bArr[i + 0] = (byte) (i2 >> 8);
    }

    public static void int32_to_buf(byte[] bArr, int i, int i2) {
        bArr[i + 3] = (byte) (i2 >> 0);
        bArr[i + 2] = (byte) (i2 >> 8);
        bArr[i + 1] = (byte) (i2 >> 16);
        bArr[i + 0] = (byte) (i2 >> 24);
    }

    public static void int64_to_buf(byte[] bArr, int i, long j) {
        bArr[i + 7] = (byte) (j >> 0);
        bArr[i + 6] = (byte) (j >> 8);
        bArr[i + 5] = (byte) (j >> 16);
        bArr[i + 4] = (byte) (j >> 24);
        bArr[i + 3] = (byte) (j >> 32);
        bArr[i + 2] = (byte) (j >> 40);
        bArr[i + 1] = (byte) (j >> 48);
        bArr[i + 0] = (byte) (j >> 56);
    }

    public static void int64_to_buf32(byte[] bArr, int i, long j) {
        bArr[i + 3] = (byte) (j >> 0);
        bArr[i + 2] = (byte) (j >> 8);
        bArr[i + 1] = (byte) (j >> 16);
        bArr[i + 0] = (byte) (j >> 24);
    }

    public static void int8_to_buf(byte[] bArr, int i, int i2) {
        bArr[i + 0] = (byte) (i2 >> 0);
    }

    public static boolean isFileExist(String str) {
        try {
            return new File(str).exists();
        } catch (Exception e) {
            return false;
        }
    }

    public static boolean isMQQExist(Context context) {
        if (context == null) {
            return false;
        }
        return true;
    }

    public static boolean isPackageExist(Context context, String str) {
        if (context == null) {
            return false;
        }
        return true;
    }

    public static boolean isTimeOutRet(int i) {
        return i == 10 || i == 161 || i == 162 || i == 164 || i == 165 || i == 166 || i == 154 || (i >= 128 && i <= 143);
    }

    @Deprecated
    public static boolean isWtLoginUrlV1(String str) {
        int indexOf;
        int i;
        int i2;
        if (str == null || (indexOf = str.indexOf("?k=")) == -1 || (i2 = (i = indexOf + 3) + 32) > str.length()) {
            return false;
        }
        String substring = str.substring(i, i2);
        return base64_decode_url(substring.getBytes(), substring.length()) != null;
    }

    public static boolean isWtLoginUrlV2(String str) {
        if (str == null) {
            return false;
        }
        String str2 = null;
        try {
            str2 = Uri.parse(str).getHost();
        } catch (Exception e) {
            printException(e);
        }
        if (!WT_LOGIN_URL_HOST.equals(str2)) {
            return false;
        }
        return isWtLoginUrlV1(str);
    }

    public static boolean is_wap_proxy_retry(Context context) {
        try {
            String str = get_apn_string(context);
            if (str == null) {
                return false;
            }
            if (!str.equalsIgnoreCase("cmwap") && !str.equalsIgnoreCase("uniwap") && !str.equalsIgnoreCase("ctwap")) {
                return str.equalsIgnoreCase("3gwap");
            }
            return true;
        } catch (Throwable th) {
            return false;
        }
    }

    public static boolean is_wap_retry(Context context) {
        return get_net_retry_type(context) != 0;
    }

    public static boolean loadLibrary(String r7, Context r8) {
        return true;
    }

    public static boolean needChangeGuid(Context context) {
        return false;
    }

    public static Bundle packBundle(byte[][] bArr) {
        Bundle bundle = new Bundle();
        if (bArr != null && bArr.length > 0) {
            bundle.putInt("len", bArr.length);
            for (int i = 0; i < bArr.length; i++) {
                bundle.putByteArray(String.valueOf(i), bArr[i]);
            }
        }
        if (bundle.isEmpty()) {
            return null;
        }
        return bundle;
    }

    public static String printByte(byte[] bArr) {
        return bArr == null ? "null" : String.valueOf(bArr.length);
    }

    public static void printException(Exception exc) {
        StringWriter stringWriter = new StringWriter();
        PrintWriter printWriter = new PrintWriter((Writer) stringWriter, true);
        exc.printStackTrace(printWriter);
        printWriter.flush();
        stringWriter.flush();
        LOGW("exception:", stringWriter.toString());
    }

    public static void printException(Exception exc, String str) {
        StringWriter stringWriter = new StringWriter();
        PrintWriter printWriter = new PrintWriter((Writer) stringWriter, true);
        exc.printStackTrace(printWriter);
        printWriter.flush();
        stringWriter.flush();
        LOGW("exception", stringWriter.toString(), str);
    }

    public static void printThrowable(Throwable th, String str) {
        StringWriter stringWriter = new StringWriter();
        PrintWriter printWriter = new PrintWriter((Writer) stringWriter, true);
        th.printStackTrace(printWriter);
        printWriter.flush();
        stringWriter.flush();
        LOGW("throwable", stringWriter.toString(), str);
    }

    public static byte[] readFile(String str) {
        if (str != null && str.length() != 0) {
            try {
                File file = new File(str);
                if (file.exists() && file.isFile()) {
                    FileInputStream fileInputStream = new FileInputStream(str);
                    int available = fileInputStream.available();
                    if (available > 528384) {
                        fileInputStream.close();
                        return null;
                    }
                    byte[] bArr = new byte[available];
                    fileInputStream.read(bArr);
                    fileInputStream.close();
                    return bArr;
                }
            } catch (Exception e) {
            }
        }
        return null;
    }

    public static byte[] readLog(Context context, String str) {
        if (context == null || str == null || str.length() == 0) {
            return null;
        }
        return readFile(getLogFileName(context, str));
    }

    public static void saveGuidToFile(Context context, byte[] bArr) {
    }

    public static void saveInitKeyTime(Context context, int i) {
    }

    public static void save_android_id(Context context, byte[] bArr) {
        if (context == null || bArr == null) {
            return;
        }
        int length = bArr.length;
    }

    public static void save_cost_time(Context context, String str) {
    }

    public static void save_cost_trace(Context context, String str) {
    }

    public static void save_cur_flag(Context context, int i) {
    }

    public static void save_cur_guid(Context context, byte[] bArr) {
        if (context == null || bArr == null) {
            return;
        }
        int length = bArr.length;
    }

    public static void save_cur_mac(Context context, byte[] bArr) {
        if (context == null || bArr == null) {
            return;
        }
        int length = bArr.length;
    }

    public static void save_network_type(Context context, int i) {
    }

    public static void set_net_retry_type(Context context, int i) {
    }

    public static byte[] string_to_buf(String str) {
        if (str == null) {
            return new byte[0];
        }
        byte[] bArr = new byte[str.length() / 2];
        for (int i = 0; i < str.length() / 2; i++) {
            int i2 = i * 2;
            bArr[i] = (byte) ((get_char((byte) str.charAt(i2)) << 4) + get_char((byte) str.charAt(i2 + 1)));
        }
        return bArr;
    }

    private static final char[] HEX_CHAR_TABLE = {'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'};

    public static String byteArrayToHexString(byte[] byteArray) {
        if (byteArray == null) {
            return null;
        }
        StringBuilder sb = new StringBuilder();
        for (byte b : byteArray) {
            char[] cArr = HEX_CHAR_TABLE;
            sb.append(cArr[(b & 240) >> 4]);
            sb.append(cArr[b & Ascii.SI]);
        }
        return sb.toString().toLowerCase();
    }
}
