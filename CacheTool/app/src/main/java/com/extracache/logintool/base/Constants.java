package com.extracache.cachetool.base;

/**
 * 常量定义类
 */
public class Constants {
    
    // 应用包名
    public static final String QQ_PACKAGE_NAME = "com.tencent.mobileqq";
    public static final String TIM_PACKAGE_NAME = "com.tencent.tim";
    public static final String APP_PACKAGE_NAME = "com.extracache.cachetool";
    
    // 文件路径
    public static final String QQ_DATA_PATH = "/data/data/" + QQ_PACKAGE_NAME;
    public static final String TIM_DATA_PATH = "/data/data/" + TIM_PACKAGE_NAME;
    public static final String APP_DATA_PATH = "/data/data/" + APP_PACKAGE_NAME;
    
    // QQ相关文件
    public static final String WLOGIN_DEVICE_FILE = "wlogin_device.dat";
    public static final String TK_FILE = "tk_file";
    public static final String MOBILE_QQ_XML = "mobileQQ.xml";
    public static final String UID_FOLDER = "uid";
    public static final String USER_FOLDER = "user";
    public static final String MMKV_FOLDER = "mmkv";
    
    // QQ相关文件（续）
    public static final String UIFA_XML = "uifa.xml";

    // 文件完整路径
    public static final String QQ_WLOGIN_DEVICE_PATH = QQ_DATA_PATH + "/files/" + WLOGIN_DEVICE_FILE;
    public static final String QQ_TK_FILE_PATH = QQ_DATA_PATH + "/databases/" + TK_FILE;
    public static final String QQ_MOBILE_XML_PATH = QQ_DATA_PATH + "/shared_prefs/" + MOBILE_QQ_XML;
    public static final String QQ_UIFA_XML_PATH = QQ_DATA_PATH + "/shared_prefs/" + UIFA_XML;
    public static final String QQ_UID_PATH = QQ_DATA_PATH + "/files/" + UID_FOLDER;
    
    // 本地文件路径（应用内部存储）
    public static final String LOCAL_WLOGIN_DEVICE_PATH = APP_DATA_PATH + "/files/" + WLOGIN_DEVICE_FILE;
    public static final String LOCAL_TK_FILE_PATH = APP_DATA_PATH + "/files/" + TK_FILE;
    public static final String LOCAL_MOBILE_XML_PATH = APP_DATA_PATH + "/files/" + MOBILE_QQ_XML;
    public static final String LOCAL_UID_PATH = APP_DATA_PATH + "/files/" + UID_FOLDER;
    
    // HTTP服务器
    public static final int DEFAULT_SERVER_PORT = 9091;
    public static final String SERVER_TAG = "QQSessionServer";
    
    // API路径
    public static final String API_CHANGE_GUID = "/changeGuid";
    public static final String API_QQ_LOGIN = "/qqlogin";
    public static final String API_QQ_SAVE = "/qqsave";
    public static final String API_QQ_TEST = "/qqtest";
    public static final String API_QQ_TIM = "/qqtim";
    public static final String API_IMPORT = "/import";
    
    // 日志标签
    public static final String LOG_TAG = "QQSessionManager";
    public static final String LOG_TAG_HTTP = "QQSessionHTTP";
    public static final String LOG_TAG_FILE = "QQSessionFile";
    public static final String LOG_TAG_CRYPTO = "QQSessionCrypto";
    public static final String LOG_TAG_COMMAND = "QQSessionCommand";
    
    // 错误码
    public static final String ERROR_FILE_NOT_FOUND = "FILE_NOT_FOUND";
    public static final String ERROR_PERMISSION_DENIED = "PERMISSION_DENIED";
    public static final String ERROR_CRYPTO_FAILED = "CRYPTO_FAILED";
    public static final String ERROR_COMMAND_FAILED = "COMMAND_FAILED";
    public static final String ERROR_INVALID_DATA = "INVALID_DATA";
    public static final String ERROR_NETWORK_ERROR = "NETWORK_ERROR";
    
    // 默认值
    public static final String DEFAULT_GUID = "D7ABE0887FFDA57040F0597663E9D773";
    public static final String NEED_KEY_STR = "client_static_keypair";
    
    // 响应消息
    public static final String MSG_SUCCESS = "操作成功";
    public static final String MSG_FAILED = "操作失败";
    public static final String MSG_FILE_NOT_FOUND = "文件未找到";
    public static final String MSG_PERMISSION_DENIED = "权限不足";
    public static final String MSG_INVALID_PARAMETER = "参数无效";
    
    private Constants() {
        // 防止实例化
    }
}
