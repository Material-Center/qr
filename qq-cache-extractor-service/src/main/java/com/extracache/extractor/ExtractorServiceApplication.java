package com.extracache.extractor;

import com.sun.net.httpserver.Headers;
import com.sun.net.httpserver.HttpExchange;
import com.sun.net.httpserver.HttpServer;

import java.io.ByteArrayOutputStream;
import java.io.IOException;
import java.io.OutputStream;
import java.net.InetSocketAddress;
import java.net.URLDecoder;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Path;
import java.nio.file.Paths;
import java.util.HashMap;
import java.util.LinkedHashMap;
import java.util.Map;
import java.util.concurrent.Executors;

public final class ExtractorServiceApplication {
    private static final int DEFAULT_PORT = 19091;

    private ExtractorServiceApplication() {
    }

    public static void main(String[] args) throws Exception {
        Map<String, String> options = parseArgs(args);
        if (options.containsKey("input")) {
            runCli(options);
            return;
        }

        int port = Integer.parseInt(options.getOrDefault("port", System.getenv().getOrDefault("PORT", String.valueOf(DEFAULT_PORT))));
        HttpServer server = HttpServer.create(new InetSocketAddress(port), 0);
        server.createContext("/health", ExtractorServiceApplication::handleHealth);
        server.createContext("/extract", ExtractorServiceApplication::handleExtract);
        server.setExecutor(Executors.newCachedThreadPool());
        server.start();
        System.out.println("qq-cache-extractor-service started, port=" + port);
    }

    private static void runCli(Map<String, String> options) throws Exception {
        ExtractResult result = new CacheExtractor().extract(Paths.get(options.get("input")), options);
        String json = result.toJson();
        if (options.containsKey("output")) {
            Files.write(Paths.get(options.get("output")), json.getBytes(StandardCharsets.UTF_8));
        } else {
            System.out.println(json);
        }
    }

    private static void handleHealth(HttpExchange exchange) throws IOException {
        send(exchange, 200, "{\"status\":\"success\",\"message\":\"ok\"}");
    }

    private static void handleExtract(HttpExchange exchange) throws IOException {
        if (!"POST".equalsIgnoreCase(exchange.getRequestMethod()) && !"GET".equalsIgnoreCase(exchange.getRequestMethod())) {
            send(exchange, 405, "{\"status\":\"error\",\"message\":\"仅支持GET/POST\"}");
            return;
        }

        Path uploadedZip = null;
        try {
            Map<String, String> options = new LinkedHashMap<>();
            options.putAll(parseQuery(exchange.getRequestURI().getRawQuery()));
            byte[] body = readAll(exchange);

            String inputPath = options.get("inputPath");
            String contentType = firstHeader(exchange.getRequestHeaders(), "Content-Type");
            if (body.length > 0 && contentType != null && contentType.toLowerCase().contains("application/json")) {
                options.putAll(JsonUtils.parseFlatObject(new String(body, StandardCharsets.UTF_8)));
                inputPath = options.get("inputPath");
            }

            Path input;
            if (inputPath != null && !inputPath.trim().isEmpty()) {
                input = Paths.get(inputPath.trim());
            } else if (body.length > 0) {
                uploadedZip = Files.createTempFile("qq-cache-upload-", ".zip");
                Files.write(uploadedZip, body);
                input = uploadedZip;
            } else {
                send(exchange, 400, errorJson("缺少inputPath或zip请求体"));
                return;
            }

            ExtractResult result = new CacheExtractor().extract(input, options);
            send(exchange, 200, result.toJson());
        } catch (Exception e) {
            send(exchange, 500, errorJson(e.getMessage()));
        } finally {
            if (uploadedZip != null) {
                Files.deleteIfExists(uploadedZip);
            }
        }
    }

    private static byte[] readAll(HttpExchange exchange) throws IOException {
        try (ByteArrayOutputStream output = new ByteArrayOutputStream()) {
            byte[] buffer = new byte[8192];
            int n;
            while ((n = exchange.getRequestBody().read(buffer)) != -1) {
                output.write(buffer, 0, n);
            }
            return output.toByteArray();
        }
    }

    private static void send(HttpExchange exchange, int status, String body) throws IOException {
        byte[] bytes = body.getBytes(StandardCharsets.UTF_8);
        exchange.getResponseHeaders().set("Content-Type", "application/json; charset=utf-8");
        exchange.sendResponseHeaders(status, bytes.length);
        try (OutputStream output = exchange.getResponseBody()) {
            output.write(bytes);
        }
    }

    private static String errorJson(String message) {
        return "{\"status\":\"error\",\"message\":" + JsonUtils.quote(message == null ? "unknown error" : message) + "}";
    }

    private static Map<String, String> parseArgs(String[] args) {
        Map<String, String> options = new HashMap<>();
        for (String arg : args) {
            if (arg == null || !arg.startsWith("--")) {
                continue;
            }
            int index = arg.indexOf('=');
            if (index > 2) {
                options.put(arg.substring(2, index), arg.substring(index + 1));
            } else {
                options.put(arg.substring(2), "true");
            }
        }
        return options;
    }

    private static Map<String, String> parseQuery(String query) throws IOException {
        Map<String, String> result = new LinkedHashMap<>();
        if (query == null || query.trim().isEmpty()) {
            return result;
        }
        String[] pairs = query.split("&");
        for (String pair : pairs) {
            int index = pair.indexOf('=');
            if (index <= 0) {
                continue;
            }
            String key = URLDecoder.decode(pair.substring(0, index), "UTF-8");
            String value = URLDecoder.decode(pair.substring(index + 1), "UTF-8");
            result.put(key, value);
        }
        return result;
    }

    private static String firstHeader(Headers headers, String name) {
        if (headers == null || headers.get(name) == null || headers.get(name).isEmpty()) {
            return "";
        }
        return headers.get(name).get(0);
    }
}
