#include <alsa/asoundlib.h>
#include <stdlib.h>
#include <string.h>
typedef struct alsa_handle alsa_handle_t;
static int card_id = -1;
static int mode = 1;

struct alsa_handle {
    snd_pcm_t* pcm_handle;
    unsigned int sample_rate;
    int channels;
};

int alsa_send(void* handle_v, const void* data, size_t bytes) {
    alsa_handle_t* handle = (alsa_handle_t*) handle_v;
    snd_pcm_sframes_t frames_written;
    size_t frames = bytes / (2 * handle->channels); // 16-bit samples

    frames_written = snd_pcm_writei(handle->pcm_handle, data, frames);

    if (frames_written < 0) {
        if (frames_written == -EPIPE) {
            fprintf(stderr, "ALSA underrun occurred on device\n");
            snd_pcm_prepare(handle->pcm_handle);
            return 0; // Return 0 to indicate underrun
        }
        fprintf(stderr, "ALSA write error on device %s\n",
                snd_strerror(frames_written));
        return -1; // Error
    }

    return (int)(frames_written * 2 * handle->channels); // Return bytes written
}

void alsa_close(void* handle_v) {
    alsa_handle_t* handle = (alsa_handle_t*) handle_v;
    if (handle) {
        if (handle->pcm_handle) {
            snd_pcm_drain(handle->pcm_handle);
            snd_pcm_close(handle->pcm_handle);
        }
        free(handle);
    }
}

void* alsa_open(unsigned int sample_rate, int channels) {
    snd_pcm_t* pcm_handle = NULL;
    int err;

    char card_name[16];
    snprintf(card_name, sizeof(card_name), "hw:%d,0", card_id);

    // Open specific hardware device (card 1, device 0)
    if ((err = snd_pcm_open(&pcm_handle, card_name, SND_PCM_STREAM_PLAYBACK, 0)) < 0) {
        fprintf(stderr, "ALSA open error: %s\n", snd_strerror(err));
        return NULL;
    }

    // Configure hardware parameters carefully
    snd_pcm_hw_params_t *params;
    snd_pcm_hw_params_alloca(&params);

    // Fill params with configuration space
    if ((err = snd_pcm_hw_params_any(pcm_handle, params)) < 0) {
        fprintf(stderr, "Cannot configure parameters: %s\n", snd_strerror(err));
        goto error;
    }

    // Set parameters in this order:
    // 1. Interleaved mode
    if ((err = snd_pcm_hw_params_set_access(pcm_handle, params, SND_PCM_ACCESS_RW_INTERLEAVED)) < 0) {
        fprintf(stderr, "Cannot set access type: %s\n", snd_strerror(err));
        goto error;
    }

    // 2. Signed 16-bit little-endian format
    if ((err = snd_pcm_hw_params_set_format(pcm_handle, params, SND_PCM_FORMAT_S16_LE)) < 0) {
        fprintf(stderr, "Cannot set format: %s\n", snd_strerror(err));
        goto error;
    }

    // 3. Channels
    if ((err = snd_pcm_hw_params_set_channels(pcm_handle, params, channels)) < 0) {
        fprintf(stderr, "Cannot set channels: %s\n", snd_strerror(err));
        goto error;
    }

    // 4. Sample rate (use nearest available)
    unsigned int actual_rate = sample_rate;
    if ((err = snd_pcm_hw_params_set_rate_near(pcm_handle, params, &actual_rate, 0)) < 0) {
        fprintf(stderr, "Cannot set rate: %s\n", snd_strerror(err));
        goto error;
    }

    // 5. Apply parameters
    if ((err = snd_pcm_hw_params(pcm_handle, params)) < 0) {
        fprintf(stderr, "Cannot set parameters: %s\n", snd_strerror(err));
        goto error;
    }

    // Create handle
    alsa_handle_t* handle = malloc(sizeof(alsa_handle_t));
    if (!handle) goto error;

    handle->pcm_handle = pcm_handle;
    handle->sample_rate = actual_rate; // Use actual supported rate
    handle->channels = channels;

    fprintf(stderr, "ALSA initialized: rate=%u, channels=%d\n", actual_rate, channels);
    return handle;

error:
    if (pcm_handle) snd_pcm_close(pcm_handle);
    return NULL;
}

int get_card_id() {
    int card_id = -1;
    int max_id = -1;
    if (snd_card_next(&card_id) < 0) { return -1; }
    if (card_id > max_id) max_id = card_id;
    if (snd_card_next(&card_id) < 0) { return -1; }
    if (card_id > max_id) max_id = card_id;
    return max_id;
}

int get_alsa_volume() {
    if (card_id < 0) { return -1; }
    snd_mixer_t *handle;
    snd_mixer_selem_id_t *sid;
    snd_mixer_elem_t *elem;
    long min, max;
    long volume;
    int err;
    const char *control_name;
    char card_name[16];

    // Determine control name based on mode
    switch(mode) {
        case 1:
            control_name = "Headphone";
            break;
        case 2:
            control_name = "Speaker";
            break;
        case 0:
        default:
            control_name = "Master";
            break;
    }

    // Create card name (e.g., "hw:1")
    snprintf(card_name, sizeof(card_name), "hw:%d", card_id);

    // Open the mixer
    if ((err = snd_mixer_open(&handle, 0)) < 0) {
        return -1;
    }

    // Attach to specified card
    if ((err = snd_mixer_attach(handle, card_name)) < 0) {
        snd_mixer_close(handle);
        // Try with "default" if specific card fails
        if ((err = snd_mixer_attach(handle, "default")) < 0) {
            snd_mixer_close(handle);
            return -1;
        }
    }

    // Register mixer
    if ((err = snd_mixer_selem_register(handle, NULL, NULL)) < 0) {
        snd_mixer_close(handle);
        return -1;
    }

    // Load mixer elements
    if ((err = snd_mixer_load(handle)) < 0) {
        snd_mixer_close(handle);
        return -1;
    }

    // Setup simple element id
    snd_mixer_selem_id_alloca(&sid);
    snd_mixer_selem_id_set_index(sid, 0);
    snd_mixer_selem_id_set_name(sid, control_name);

    // Find the element
    elem = snd_mixer_find_selem(handle, sid);
    if (!elem) {
        snd_mixer_close(handle);
        return -1;
    }

    // Get volume range
    snd_mixer_selem_get_playback_volume_range(elem, &min, &max);

    // Get playback volume (try front left first)
    if (snd_mixer_selem_get_playback_volume(elem, SND_MIXER_SCHN_FRONT_LEFT, &volume) < 0) {
        // If front left fails, try mono
        if (snd_mixer_selem_get_playback_volume(elem, SND_MIXER_SCHN_MONO, &volume) < 0) {
            snd_mixer_close(handle);
            return -1;
        }
    }

    snd_mixer_close(handle);

    // Convert to percentage
    int percent = (int)((double)(volume - min) / (max - min) * 100.0);
    
    // Ensure percentage is within bounds
    if (percent < 0) return 0;
    if (percent > 100) return 100;
    return percent;
}

int set_alsa_volume(int percentage) {
    if (card_id < 0) { return -1; }
    snd_mixer_t *handle;
    snd_mixer_selem_id_t *sid;
    snd_mixer_elem_t *elem;
    long min, max;
    int err;
    const char *control_name;
    char card_name[16];

    // Clamp percentage
    if (percentage < 0) percentage = 0;
    if (percentage > 100) percentage = 100;

    // Determine control name based on mode
    switch(mode) {
        case 1:
            control_name = "Headphone";
            break;
        case 2:
            control_name = "Speaker";
            break;
        case 0:
        default:
            control_name = "Master";
            break;
    }

    // Create card name (e.g., "hw:1")
    snprintf(card_name, sizeof(card_name), "hw:%d", card_id);

    // Open the mixer
    if ((err = snd_mixer_open(&handle, 0)) < 0) {
        return -1;
    }

    // Attach to specified card
    if ((err = snd_mixer_attach(handle, card_name)) < 0) {
        snd_mixer_close(handle);
        // Try with "default" if specific card fails
        if ((err = snd_mixer_attach(handle, "default")) < 0) {
            snd_mixer_close(handle);
            return -1;
        }
    }

    // Register mixer
    if ((err = snd_mixer_selem_register(handle, NULL, NULL)) < 0) {
        snd_mixer_close(handle);
        return -1;
    }

    // Load mixer elements
    if ((err = snd_mixer_load(handle)) < 0) {
        snd_mixer_close(handle);
        return -1;
    }

    // Setup simple element id
    snd_mixer_selem_id_alloca(&sid);
    snd_mixer_selem_id_set_index(sid, 0);
    snd_mixer_selem_id_set_name(sid, control_name);

    // Find the element
    elem = snd_mixer_find_selem(handle, sid);
    if (!elem) {
        snd_mixer_close(handle);
        return -1;
    }

    // Get volume range
    snd_mixer_selem_get_playback_volume_range(elem, &min, &max);

    // Calculate and set new volume
    long new_vol = min + (long)((max - min) * percentage / 100.0);

    // Set volume for all channels
    if (snd_mixer_selem_set_playback_volume_all(elem, new_vol) < 0) {
        snd_mixer_close(handle);
        return -1;
    }

    snd_mixer_close(handle);
    return percentage;
}

// Toggle mute (1=mute, 0=unmute, -1=toggle)
int set_alsa_mute(int mode, int mute_action) {
    if (card_id < 0) { return -1; }
    snd_mixer_t *handle;
    snd_mixer_selem_id_t *sid;
    snd_mixer_elem_t *elem;
    int err, current, new_state;
    const char *control_name;
    char card_name[16];

    switch(mode) {
        case 1: control_name = "Headphone"; break;
        case 2: control_name = "Speaker"; break;
        default: control_name = "Master"; break;
    }

    snprintf(card_name, sizeof(card_name), "hw:%d", card_id);

    if ((err = snd_mixer_open(&handle, 0)) < 0) return -1;

    do {
        if ((err = snd_mixer_attach(handle, card_name)) < 0 &&
            (err = snd_mixer_attach(handle, "default")) < 0) break;

        if ((err = snd_mixer_selem_register(handle, NULL, NULL)) < 0) break;
        if ((err = snd_mixer_load(handle)) < 0) break;

        snd_mixer_selem_id_alloca(&sid);
        snd_mixer_selem_id_set_index(sid, 0);
        snd_mixer_selem_id_set_name(sid, control_name);

        if (!(elem = snd_mixer_find_selem(handle, sid))) break;

        // Get current mute state
        if (snd_mixer_selem_get_playback_switch(elem, SND_MIXER_SCHN_FRONT_LEFT, &current) < 0) break;

        // Determine new state
        if (mute_action == -1) { // Toggle
            new_state = !current;
        } else { // Set directly
            new_state = mute_action ? 0 : 1; // Invert because ALSA uses 0=muted
        }

        // Set mute state for all channels
        if (snd_mixer_selem_set_playback_switch_all(elem, new_state) < 0) break;

        snd_mixer_close(handle);
        return new_state ? 0 : 1; // Return 1=muted, 0=unmuted
    } while(0);

    snd_mixer_close(handle);
    return -1;
}

void switch_alsa_mode() {
    set_alsa_mute(mode, 1);
    mode = 3 - mode;
    set_alsa_mute(mode, 0);
}

void init_alsa() {
    card_id = get_card_id();
    if (card_id >= 0) {
        set_alsa_mute(0, 0);
        mode = 0;
        set_alsa_volume(99);
        mode = 1;
    }
}
