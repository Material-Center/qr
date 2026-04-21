package com.extracache.cachetool.test;

import android.util.Log;

import com.extracache.cachetool.model.SessionData;
import com.extracache.cachetool.service.Ini4jParser;

/**
 * INI解析器测试类
 * 用于测试解析你提供的INI文件格式
 */
public class IniParserTest {
    private static final String TAG = "IniParserTest";
    
    // 你提供的测试INI内容
    private static final String TEST_INI_CONTENT = 
        "[3890637088]\n" +
        "qqnum=3890637088\n" +
        "guid=F950BBC31CF28FA92E034DF15B43D9B2\n" +
        "uid=u_oT4DGwxTrchYc7wIhj2U_w\n" +
        "Token010E=5F712929494D375A506D5748612A4E38\n" +
        "_psKey=000F000D6F66666963652E71712E636F6D002C626C68315564567052555544562D30774A59375158766C2D31444E6C58334F44684F33552D786A65752A6B5FFFFF0000000068C306DE000A71756E2E71712E636F6D002C5A68435376656656616D5A43656F47326C2D7348505976537154377575615169304E494E567474713854345FFFFF0000000068C306DE001167616D6563656E7465722E71712E636F6D002C684E7070564B6744415743636F38486E4661776934596551556E7168546C5A42544E6F7370545072747A515FFFFF0000000068C306DE000B646F63732E71712E636F6D002C4C534739653661486E53442A756B4D2D434E367A377A5446336874793956786141474D57584664576E66495FFFFF0000000068C306DE000B6D61696C2E71712E636F6D002C412D3242343842687239636F4758497950526C6B46702A7938524A47446447483863497252722A62556E595FFFFF0000000068C306DE000A74696D2E71712E636F6D002C7A4D615130302D777464546F46646E487A556D68363177592D30536A2D6D4F457264647048306C663732735FFFFF0000000068C306DE000974692E71712E636F6D002C725750656D6B50696E516C59584A3878317764766B773146446544656855367937654668384832734767595FFFFF0000000068C306DE000A7669702E71712E636F6D002C65372D6A666E7157422D7369696E2A53706461725572667858413241456B70634C646932436630594366675FFFFF0000000068C306DE000A74656E7061792E636F6D002C49315770547450634431556D6F434D636350742D717A3463766D454942306F65706C5A3439346A4D38736F5FFFFF0000000068C306DE000C71717765622E71712E636F6D002C5236634F46585150744747534F4C4D6B6E6E7742444F4D516663434E5052635434706E786952665A6F4E495FFFFF0000000068C306DE000C717A6F6E652E71712E636F6D002C392D4445464D514C2D612D716D424E2A476B33767930782D3043454754474136713857616C77776F3145735FFFFF0000000068C306DE000A6D6D612E71712E636F6D002C5A673936797262335A5A6D757269564259714B4B31486A5443346D346B6F3565524563527A554A767575635FFFFF0000000068C306DE000B67616D652E71712E636F6D002C612A636A63485A2D5043516D3939662D6134307658646F4D4845715A3349674738596B44637569785230385FFFFF0000000068C306DE00116F70656E6D6F62696C652E71712E636F6D002C4A755344307A7636796F4E327052654F69553370445964314261484B3771717343434359583876527966735FFFFF0000000068C306DE000E636F6E6E6563742E71712E636F6D002C4867567A715032776535696F36692D6B42363558583143454A48387842436752307679542D6839332D38385FFFFF0000000068C306DE\n" +
        "sessionKey=715E3254646153412D23483628777149\n" +
        "Token010A=F5D64B757745657F5E1F44C000C3258011937FC39C570ECDF983D42400FBC24ACF5C82BC5037E198C1D21191AAF98ABEBAA6156CDB9A6C911ADE94152CB318D6D31CCCE63EE4F8F8\n" +
        "_randseed=B465B2DA40994DB7\n" +
        "_G=8EAE5F5ED06626EA5CC236C59E28E993\n" +
        "Token016A=368D7C0D0FBCB91609F486D70EE07AF3532FD54DCA439C04460FEC31136FF4EB37240B8D214FD5763BA7F38E3C8BC4DA9867C3AC43BA45DA4E551F2AFC98C9BF85EFABF946F404A2F3CE7ED6871173C3\n" +
        "_superKey=6662632D59376344554561575446644A4F413148597A7679496F4D5A542A5174544B4D436574364C6C6A415F\n" +
        "_userStWebSig=9D80FFA7468D75506792D01A720049921583E4153FBC73AFAAED76DE86C1E75F9A3936AB69AFDB976187D543687C69E8\n" +
        "Token0106=9586465E7B09214685880A3677C18C3AC973A691AA547E374B5C1E386AB2778E37153250F46E55AB5CB318963A7295EDCFF67F639B8EB996C8600A7DAB85B7CA977F9E42385495BEEAE176893EFCD337A537281D5BA948848CCBDAD9A860D642476C369241C1A6A2E397BDBF1904883F091842350BEA0475FEC7E353BC71B9DED585C0923519A9C89F1D41E7B789759139A08FF78C686C8F9A47BACB813B6222\n" +
        "_dpwd=415749586870454C61646C6B6E6E676D\n" +
        "Token0114=000168C1B55E00588A251F1D3CA6F95FAF7D1D273BD20BEB931051383CB1FFAA6FD33B316B0AEFB84AB6C7517B136F41610AD8D4A7F2F484C96CBD4F2DA5E9D61AAA0919F885E18E7166D0264B6B48D6C7EA36A84823C8A98B144F26C5D4EE52\n" +
        "Token0134=EE278047D763DE9694F0DE42D30F9CBD\n" +
        "Token0133=B99180AEFA72A86336C12BAAD8A2CB4E4E46CA51D6907B088066FB5C17931AEF1ED4FEBF7D1CA5076EC58CE9D5709990\n" +
        "Token0143=3577823C99E649D784D73B40156F686EC1F606D0D8126C5442BA07D539D56636A7E8ADBDAE51FF563C369216E46CF14BF0036A9995894B226BBEEE33D861393747C9F487C7516AA02E35A11239C04D462026BA1BE19E9C25\n" +
        "TGTKey=7E2B686D6D6B5A385E5D3E4D35244343\n" +
        "_sKey=4D34454941696C6B3838\n" +
        "extractTime=1757526689911\n" +
        "deviceInfo=品牌: HONOR型号: ATH-AL00Android版本: 8.1.0API级别: 27序列号: 67A3Q1571043osrg";
    
    /**
     * 测试解析你提供的INI格式
     */
    public static void testParseIniFormat() {
        Log.d(TAG, "开始测试INI解析器...");
        
        try {
            SessionData sessionData = Ini4jParser.parseIniToSessionData(TEST_INI_CONTENT);
            
            // 验证基本字段
            Log.d(TAG, "=== 基本字段验证 ===");
            Log.d(TAG, "QQ号: " + sessionData.getQq());
            Log.d(TAG, "UID: " + sessionData.getUid());
            Log.d(TAG, "GUID: " + sessionData.getGuid());
            
            // 验证关键Token
            Log.d(TAG, "=== 关键Token验证 ===");
            Log.d(TAG, "sessionKey: " + (sessionData.getSessionKey() != null ? "存在" : "缺失"));
            Log.d(TAG, "Token0143: " + (sessionData.getToken0143() != null ? "存在" : "缺失"));
            Log.d(TAG, "Token010A: " + (sessionData.getToken010A() != null ? "存在" : "缺失"));
            Log.d(TAG, "Token0106: " + (sessionData.getToken0106() != null ? "存在" : "缺失"));
            Log.d(TAG, "TGTKey: " + (sessionData.getTgtKey() != null ? "存在" : "缺失"));
            Log.d(TAG, "Token0133: " + (sessionData.getToken0133() != null ? "存在" : "缺失"));
            Log.d(TAG, "Token0134: " + (sessionData.getToken0134() != null ? "存在" : "缺失"));
            Log.d(TAG, "Token016A: " + (sessionData.getToken016A() != null ? "存在" : "缺失"));
            
            // 验证其他Token
            Log.d(TAG, "=== 其他Token验证 ===");
            Log.d(TAG, "_sKey: " + (sessionData.getSKey() != null ? "存在" : "缺失"));
            Log.d(TAG, "_psKey: " + (sessionData.getPsKey() != null ? "存在" : "缺失"));
            Log.d(TAG, "_superKey: " + (sessionData.getSuperKey() != null ? "存在" : "缺失"));
            Log.d(TAG, "_userStWebSig: " + (sessionData.getUserStWebSig() != null ? "存在" : "缺失"));
            Log.d(TAG, "_userStSig: " + (sessionData.getUserStSig() != null ? "存在" : "缺失"));
            
            // 验证特殊Token
            Log.d(TAG, "=== 特殊Token验证 ===");
            Log.d(TAG, "Token010E: " + (sessionData.getToken("Token010E") != null ? "存在" : "缺失"));
            Log.d(TAG, "Token0114: " + (sessionData.getToken("Token0114") != null ? "存在" : "缺失"));
            Log.d(TAG, "_randseed: " + (sessionData.getToken("_randseed") != null ? "存在" : "缺失"));
            Log.d(TAG, "_G: " + (sessionData.getToken("_G") != null ? "存在" : "缺失"));
            Log.d(TAG, "_dpwd: " + (sessionData.getToken("_dpwd") != null ? "存在" : "缺失"));
            
            // 验证数据有效性
            Log.d(TAG, "=== 数据有效性验证 ===");
            Log.d(TAG, "SessionData是否有效: " + sessionData.isValid());
            Log.d(TAG, "是否有基本Token: " + sessionData.hasBasicTokens());
            Log.d(TAG, "总Token数量: " + sessionData.getTokens().size());
            
            // 打印所有Token
            Log.d(TAG, "=== 所有Token列表 ===");
            for (String key : sessionData.getTokens().keySet()) {
                String value = sessionData.getToken(key);
                String displayValue = value.length() > 20 ? value.substring(0, 20) + "..." : value;
                Log.d(TAG, key + ": " + displayValue);
            }
            
            // 验证结果
            boolean isValid = sessionData.isValid() && 
                             sessionData.getQq() != null && 
                             sessionData.getQq().equals("3890637088") &&
                             sessionData.getGuid() != null &&
                             sessionData.getUid() != null;
            
            Log.d(TAG, "=== 测试结果 ===");
            Log.d(TAG, "解析" + (isValid ? "成功" : "失败"));
            
            if (isValid) {
                Log.d(TAG, "✅ 所有基本字段解析正确");
                Log.d(TAG, "✅ QQ号: " + sessionData.getQq());
                Log.d(TAG, "✅ UID: " + sessionData.getUid());
                Log.d(TAG, "✅ GUID: " + sessionData.getGuid());
                Log.d(TAG, "✅ 解析到 " + sessionData.getTokens().size() + " 个Token");
            } else {
                Log.e(TAG, "❌ 解析失败或数据不完整");
            }
            
        } catch (Exception e) {
            Log.e(TAG, "测试过程中发生错误", e);
        }
    }
    
    /**
     * 测试特定Token的解析
     */
    public static void testSpecificTokens() {
        Log.d(TAG, "开始测试特定Token解析...");
        
        SessionData sessionData = Ini4jParser.parseIniToSessionData(TEST_INI_CONTENT);
        
        // 测试你提供的具体Token值
        String expectedSessionKey = "715E3254646153412D23483628777149";
        String expectedToken0143 = "3577823C99E649D784D73B40156F686EC1F606D0D8126C5442BA07D539D56636A7E8ADBDAE51FF563C369216E46CF14BF0036A9995894B226BBEEE33D861393747C9F487C7516AA02E35A11239C04D462026BA1BE19E9C25";
        
        Log.d(TAG, "sessionKey匹配: " + expectedSessionKey.equals(sessionData.getSessionKey()));
        Log.d(TAG, "Token0143匹配: " + expectedToken0143.equals(sessionData.getToken0143()));
        
        if (expectedSessionKey.equals(sessionData.getSessionKey())) {
            Log.d(TAG, "✅ sessionKey解析正确");
        } else {
            Log.e(TAG, "❌ sessionKey解析错误");
            Log.e(TAG, "期望: " + expectedSessionKey);
            Log.e(TAG, "实际: " + sessionData.getSessionKey());
        }
        
        if (expectedToken0143.equals(sessionData.getToken0143())) {
            Log.d(TAG, "✅ Token0143解析正确");
        } else {
            Log.e(TAG, "❌ Token0143解析错误");
            Log.e(TAG, "期望: " + expectedToken0143);
            Log.e(TAG, "实际: " + sessionData.getToken0143());
        }
    }
    
    /**
     * 测试ini4j解析器的特殊功能
     */
    public static void testIni4jFeatures() {
        Log.d(TAG, "开始测试ini4j解析器特殊功能...");
        
        // 测试INI文件信息获取
        String iniInfo = Ini4jParser.getIniInfo(TEST_INI_CONTENT);
        Log.d(TAG, "INI文件信息:\n" + iniInfo);
        
        // 测试格式验证
        boolean isValid = Ini4jParser.isValidIniFormat(TEST_INI_CONTENT);
        Log.d(TAG, "INI格式验证: " + (isValid ? "有效" : "无效"));
        
        // 测试空内容
        boolean isEmptyValid = Ini4jParser.isValidIniFormat("");
        Log.d(TAG, "空内容验证: " + (isEmptyValid ? "有效" : "无效"));
        
        // 测试无效格式
        boolean isInvalidValid = Ini4jParser.isValidIniFormat("这不是INI格式");
        Log.d(TAG, "无效格式验证: " + (isInvalidValid ? "有效" : "无效"));
    }
}
