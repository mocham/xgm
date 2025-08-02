#include <stb_wrapper.h>
#include <webp/decode.h>
#include <webp/encode.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include <turbojpeg.h>
#include <math.h>
const int IMG_SUCCESS=0;
const int IMG_ERROR_LOAD=1;
const int IMG_ERROR_INVALID_PARAM=2;
const int IMG_ERROR_RESIZE=3;

int convert_rgba(uint32_t* pixels, int width, int height) {
    for (int i = 0; i < width * height; i++) {
        const uint32_t px = pixels[i];
        // Swap R and B channels while preserving G and A
        pixels[i] = (px & 0xFF00FF00) |  // Keep G and A channels
                   ((px & 0x00FF0000) >> 16) |  // R -> B
                   ((px & 0x000000FF) << 16);    // B -> R
    }
}

// Track allocation source to use correct free function
typedef enum { ALLOC_STBI, ALLOC_WEBP, ALLOC_JPEG_TURBO } alloc_type_t;

int img_load_bgra(const char *filename, unsigned char **out_pixels, int *out_width, int *out_height, alloc_type_t *alloc_type, int flip) {
    if (!filename || !out_pixels || !out_width || !out_height || !alloc_type) return IMG_ERROR_INVALID_PARAM;
    int width, height, channels;
    unsigned char *rgba;
    if (strstr(filename, ".webp")) {
        FILE *file = fopen(filename, "rb");
        if (!file) {
            printf("Cannot open file: %s\n", filename);
            return IMG_ERROR_LOAD;
        }
        fseek(file, 0, SEEK_END);
        size_t size = ftell(file);
        fseek(file, 0, SEEK_SET);
        uint8_t *data = malloc(size);
        if (!data) {
            fclose(file);
            return IMG_ERROR_LOAD;
        }
        fread(data, 1, size, file);
        fclose(file);
        if (!WebPGetInfo(data, size, &width, &height)) {
            printf("WebPGetInfo failed\n");
            free(data);
            return IMG_ERROR_LOAD;
        }
        rgba = WebPDecodeRGBA(data, size, &width, &height);
        free(data);
        if (!rgba) {
            printf("WebPDecodeRGBA failed\n");
            return IMG_ERROR_LOAD;
        }
        *out_pixels = rgba;
        if (flip) convert_rgba((uint32_t*)*out_pixels, width, height);
        *out_width = width;
        *out_height = height;
        *alloc_type = ALLOC_WEBP;
        return IMG_SUCCESS;
    } else if (strstr(filename, ".jpg") || strstr(filename, ".jpeg")) {
        printf("Use jpeg-turbo\n");
        tjhandle jpeg = tjInitDecompress();
        if (!jpeg) {
            printf("tjInitDecompress failed: %s\n", tjGetErrorStr());
            return IMG_ERROR_LOAD;
        }
        FILE *file = fopen(filename, "rb");
        if (!file) {
            printf("Cannot open file: %s\n", filename);
            tjDestroy(jpeg);
            return IMG_ERROR_LOAD;
        }
        fseek(file, 0, SEEK_END);
        size_t size = ftell(file);
        fseek(file, 0, SEEK_SET);
        unsigned char *jpegBuf = malloc(size);
        if (!jpegBuf) {
            fclose(file);
            tjDestroy(jpeg);
            return IMG_ERROR_LOAD;
        }
        fread(jpegBuf, 1, size, file);
        fclose(file);
        int width, height, jpegSubsamp, jpegColorspace;
        if (tjDecompressHeader3(jpeg, jpegBuf, size, &width, &height,
                               &jpegSubsamp, &jpegColorspace) < 0) {
            printf("tjDecompressHeader3 failed: %s\n", tjGetErrorStr());
            free(jpegBuf);
            tjDestroy(jpeg);
            return IMG_ERROR_LOAD;
        }
        unsigned char *bgra = malloc(width * height * 4);
        if (!bgra) {
            free(jpegBuf);
            tjDestroy(jpeg);
            return IMG_ERROR_LOAD;
        }
        if (tjDecompress2(jpeg, jpegBuf, size, bgra, width, 0, height,
                          TJPF_BGRA, TJFLAG_FASTDCT) < 0) {
            printf("tjDecompress2 failed: %s\n", tjGetErrorStr());
            free(jpegBuf);
            free(bgra);
            tjDestroy(jpeg);
            return IMG_ERROR_LOAD;
        }
        free(jpegBuf);
        tjDestroy(jpeg);
        *out_pixels = bgra;
        *out_width = width;
        *out_height = height;
        *alloc_type = ALLOC_JPEG_TURBO;
        return IMG_SUCCESS;
    } else {
        rgba = wrap_stbi_load(filename, &width, &height, &channels, 4);
        if (rgba) {
            *out_pixels = rgba;
            *out_width = width;
            *out_height = height;
            *alloc_type = ALLOC_STBI;
            if (flip) convert_rgba((uint32_t*)*out_pixels, width, height);
            return IMG_SUCCESS;
        }
    }

    printf("stb_image error: %s\n", wrap_stbi_failure_reason());
    return IMG_ERROR_LOAD;
}

int img_load_bgra_from_memory(const unsigned char *data, int data_len, unsigned char **out_pixels, int *out_width, int *out_height, alloc_type_t *alloc_type, int flip) {
    if (!data || data_len <= 0 || !out_pixels || !out_width || !out_height || !alloc_type) return IMG_ERROR_INVALID_PARAM;
    int width, height, channels;
    unsigned char *rgba;
    if (data_len >= 12 && memcmp(data, "RIFF", 4) == 0 && memcmp(data + 8, "WEBP", 4) == 0) {
        if (!WebPGetInfo(data, data_len, &width, &height)) {
            printf("WebPGetInfo failed\n");
            return IMG_ERROR_LOAD;
        }
        rgba = WebPDecodeRGBA(data, data_len, &width, &height);
        if (!rgba) {
            printf("WebPDecodeRGBA failed\n");
            return IMG_ERROR_LOAD;
        }
        *out_pixels = rgba;
        *out_width = width;
        *out_height = height;
        *alloc_type = ALLOC_WEBP;
        if (flip) convert_rgba((uint32_t*)*out_pixels, width, height);
        return IMG_SUCCESS;
    } else {
        rgba = wrap_stbi_load_from_memory(data, data_len, &width, &height, &channels, 4);
        if (rgba) {
            *out_pixels = rgba;
            *out_width = width;
            *out_height = height;
            *alloc_type = ALLOC_STBI;
            if (flip) convert_rgba((uint32_t*)*out_pixels, width, height);
            return IMG_SUCCESS;
        }
    }
    printf("stb_image error: %s\n", wrap_stbi_failure_reason());
    return IMG_ERROR_LOAD;
}

void img_free_buffer(void *buffer, alloc_type_t alloc_type) {
    if (buffer) {
        if (alloc_type == ALLOC_WEBP) {
            WebPFree(buffer);
        } else if (alloc_type == ALLOC_STBI) {
            wrap_stbi_image_free(buffer);
        } else {
            free(buffer);
        }
    }
}

int img_resize_bgra_to_fit(const unsigned char *in_pixels, int in_width, int in_height, int max_width, int max_height, unsigned char *output, int *out_width, int *out_height, int flip) {
    if (!in_pixels || in_width <= 0 || in_height <= 0 || max_width <= 0 || max_height <= 0 || !output || !out_width || !out_height)
        return IMG_ERROR_INVALID_PARAM;
    int new_width = in_width;
    int new_height = in_height;
    if (in_width * max_height > in_height * max_width) {
        if (in_width > max_width) {
            new_width = max_width;
            new_height = in_height * max_width / in_width;
        }
    } else {
        if (in_height > max_height) {
            new_height = max_height;
            new_width = in_width * max_height / in_height;
        }
    }
    if (new_width == in_width && new_height == in_height) {
        memcpy(output, in_pixels, in_width * in_height * 4);
        *out_width = in_width;
        *out_height = in_height;
        if (flip) convert_rgba((uint32_t*)output, new_width, new_height);
        return IMG_SUCCESS;
    }
    if (!wrap_stbir_resize_uint8_srgb(in_pixels, in_width, in_height, in_width * 4,
                                output, new_width, new_height, new_width * 4,
                                wrap_STBIR_4CHANNEL())) {
        return IMG_ERROR_RESIZE;
    }
    *out_width = new_width;
    *out_height = new_height;
    if (flip) convert_rgba((uint32_t*)output, new_width, new_height);
    return IMG_SUCCESS;
}

void center_and_extend_image(const char* input_path, const char* output_path, int target_width, int target_height) {
    // Load the original image
    int orig_width, orig_height, orig_channels;
    unsigned char* orig_data = wrap_stbi_load(input_path, &orig_width, &orig_height, &orig_channels, 0);
    if (!orig_data) {
        fprintf(stderr, "Error loading image: %s\n", wrap_stbi_failure_reason());
        return;
    }
    // Create new canvas with target dimensions
    unsigned char* new_data = (unsigned char*)malloc(target_width * target_height * orig_channels);
    if (!new_data) {
        fprintf(stderr, "Memory allocation failed\n");
        wrap_stbi_image_free(orig_data);
        return;
    }

    // Fill with transparent black (0) or white (255) depending on your needs
    memset(new_data, 0, target_width * target_height * orig_channels);

    // Calculate centered position
    int x_offset = (target_width - orig_width) / 2;
    int y_offset = (target_height - orig_height) / 2;

    // Copy original image to centered position
    for (int y = 0; y < orig_height; y++) {
        if (y + y_offset >= 0 && y + y_offset < target_height) {
            for (int x = 0; x < orig_width; x++) {
                if (x + x_offset >= 0 && x + x_offset < target_width) {
                    for (int c = 0; c < orig_channels; c++) {
                        int new_idx = ((y + y_offset) * target_width + (x + x_offset)) * orig_channels + c;
                        int orig_idx = (y * orig_width + x) * orig_channels + c;
                        new_data[new_idx] = orig_data[orig_idx];
                    }
                }
            }
        }
    }

    // Save the result
    char* ext = strrchr(output_path, '.');
    if (ext) {
        if (strcmp(ext, ".png") == 0) {
            wrap_stbi_write_png(output_path, target_width, target_height, orig_channels, new_data, target_width * orig_channels);
        } else if (strcmp(ext, ".jpg") == 0 || strcmp(ext, ".jpeg") == 0) {
            wrap_stbi_write_jpg(output_path, target_width, target_height, orig_channels, new_data, 90);
        } else {
            fprintf(stderr, "Unsupported output format\n");
        }
    } else {
        fprintf(stderr, "No extension found in output path\n");
    }

    // Clean up
    wrap_stbi_image_free(orig_data);
    free(new_data);
}

int save_png(const unsigned char* pixels, int width, int height, const char* filename) {
    return wrap_stbi_write_png(filename, width, height, 4, pixels, width * 4);
}
////////////////////////////////////////////////

char* encode_rgba_to_webp(const uint8_t* rgba, int width, int height, int stride, float quality_factor, size_t* output_size) {
    WebPPicture picture;
    WebPConfig config;
    WebPMemoryWriter writer;

    if (!WebPConfigPreset(&config, WEBP_PRESET_DEFAULT, quality_factor) ||
        !WebPPictureInit(&picture)) {
        return NULL;
    }

    picture.use_argb = 1;
    picture.width = width;
    picture.height = height;
    picture.writer = WebPMemoryWrite;
    picture.custom_ptr = &writer;
    WebPMemoryWriterInit(&writer);

    // Import RGBA data
    if (!WebPPictureImportRGBA(&picture, rgba, stride)) {
        WebPPictureFree(&picture);
        return NULL;
    }

    if (!WebPEncode(&config, &picture)) {
        WebPPictureFree(&picture);
        return NULL;
    }

    *output_size = writer.size;
    char* output = malloc(writer.size);
    if (output) {
        memcpy(output, writer.mem, writer.size);
    }

    WebPPictureFree(&picture);
    return output;
}
