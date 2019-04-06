package template

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http"
	"path"
	"strconv"
	"text/template"

	"github.com/pipelined/mp3"
	"github.com/pipelined/phono/convert"
	"github.com/pipelined/signal"
)

type convertForm struct {
	data []byte
}

// convertData provides a data for convert form, so user can define conversion parameters.
type convertData struct {
	Accept     string
	OutFormats []convert.Format
	WavOptions wavOptions
	Mp3Options mp3Options
}

// WavOptions is a struct of wav options that are available for conversion.
type wavOptions struct {
	BitDepths map[signal.BitDepth]string
}

type mp3Options struct {
	VBR           mp3.BitRateMode
	ABR           mp3.BitRateMode
	CBR           mp3.BitRateMode
	BitRateModes  map[mp3.BitRateMode]string
	ChannelModes  map[mp3.ChannelMode]string
	DefineQuality bool
}

var (
	convertTemplate = template.Must(template.New("convert").Parse(convertHTML))

	convertFormData = convertData{
		Accept: fmt.Sprintf(".%s, .%s", convert.WavFormat, convert.Mp3Format),
		OutFormats: []convert.Format{
			convert.WavFormat,
			convert.Mp3Format,
		},
		WavOptions: wavOptions{
			BitDepths: convert.Supported.WavBitDepths,
		},
		Mp3Options: mp3Options{
			VBR:          mp3.VBR,
			ABR:          mp3.ABR,
			CBR:          mp3.CBR,
			BitRateModes: convert.Supported.Mp3BitRateModes,
			ChannelModes: convert.Supported.Mp3ChannelModes,
		},
	}

	// ConvertForm is the convert form.
	ConvertForm = parseConvertForm()
)

func parseConvertForm() convertForm {
	var b bytes.Buffer
	if err := convertTemplate.Execute(&b, convertFormData); err != nil {
		panic(fmt.Sprintf("Failed to parse convert template: %v", err))
	}
	return convertForm{data: b.Bytes()}
}

// Data returns serialized form data, ready to be served.
func (cf convertForm) Data() []byte {
	return cf.data
}

// Format parses input format from http request.
func (convertForm) Format(r *http.Request) convert.Format {
	return convert.Format(path.Base(r.URL.Path))
}

// File returns multipart file from http request.
func (convertForm) File(r *http.Request) (multipart.File, *multipart.FileHeader, error) {
	return r.FormFile("input-file")
}

// Prase form data into output config.
func (convertForm) Parse(r *http.Request) (convert.OutputConfig, error) {
	f := convert.Format(r.FormValue("format"))
	switch f {
	case convert.WavFormat:
		return parseWavConfig(r)
	case convert.Mp3Format:
		return parseMp3Config(r)
	default:
		return nil, fmt.Errorf("Unsupported format: %v", f)
	}
}

func parseWavConfig(r *http.Request) (convert.WavConfig, error) {
	// try to get bit depth
	bitDepth, err := parseIntValue(r, "wav-bit-depth", "bit depth")
	if err != nil {
		return convert.WavConfig{}, err
	}

	return convert.WavConfig{BitDepth: signal.BitDepth(bitDepth)}, nil
}

func parseMp3Config(r *http.Request) (convert.Mp3Config, error) {
	// try to get bit rate mode
	bitRateMode, err := parseIntValue(r, "mp3-bit-rate-mode", "bit rate mode")
	if err != nil {
		return convert.Mp3Config{}, err
	}

	// try to get channel mode
	channelMode, err := parseIntValue(r, "mp3-channel-mode", "channel mode")
	if err != nil {
		return convert.Mp3Config{}, err
	}

	if mp3.BitRateMode(bitRateMode) == mp3.VBR {
		// try to get vbr quality
		vbrQuality, err := parseIntValue(r, "mp3-vbr-quality", "vbr quality")
		if err != nil {
			return convert.Mp3Config{}, err
		}
		return convert.Mp3Config{
			BitRateMode: mp3.VBR,
			ChannelMode: mp3.ChannelMode(channelMode),
			VBRQuality:  mp3.VBRQuality(vbrQuality),
		}, nil
	}

	// try to get bitrate
	bitRate, err := parseIntValue(r, "mp3-bit-rate", "bit rate")
	if err != nil {
		return convert.Mp3Config{}, err
	}
	return convert.Mp3Config{
		BitRateMode: mp3.BitRateMode(bitRateMode),
		ChannelMode: mp3.ChannelMode(channelMode),
		BitRate:     bitRate,
	}, nil
}

// parseIntValue parses value of key provided in the html form.
// Returns error if value is not provided or cannot be parsed as int.
func parseIntValue(r *http.Request, key, name string) (int, error) {
	str := r.FormValue(key)
	if str == "" {
		return 0, fmt.Errorf("Please provide %s", name)
	}

	val, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("Failed parsing %s %s: %v", name, str, err)
	}
	return val, nil
}

const convertHTML = `
<html>
<head>
    <style>
        * {
            font-family: Verdana;
        }
        form {
            margin: 0;
        }
        button {
            background:none!important;
            color:inherit;
            border:none; 
            padding:0!important;
            font: inherit;
            border-bottom:1px solid #444; 
            cursor: pointer;
        }
        #input-file-label {
            cursor: pointer;
            padding:0!important;
            border-bottom:1px solid #444; 
        }
        #mp3-quality {
            padding-bottom: 10px;
        }
    </style>
    <script type="text/javascript">
        document.addEventListener("DOMContentLoaded", function(event) {
            document.getElementById("convert").reset();
        });
        function getFileName(id) {
            var filePath = document.getElementById(id).value;
            return filePath.substr(filePath.lastIndexOf('\\') + 1);
        }
        function displayClass(className, display) {
            var elements = document.getElementsByClassName(className);
            for (var i = 0, ii = elements.length; i < ii; i++) {
                elements[i].style.display = display ? '' : 'none';
            };
        }
        function displayId(id, mode){
            document.getElementById(id).style.display = mode;
        }
        function onInputFileChange(){
            document.getElementById('input-file-label').innerHTML = getFileName('input-file');
            displayClass('input-file-label', true);
            displayId('output-format', "");
        }
		function onOutputFormatsClick(el){
        	displayClass('output-options', false);
        	displayId(el.id+'-options', "");
        	displayId('submit', "");
        }
        function onMp3BitRateModeChange(el){
        	displayClass('mp3-bit-rate-mode-options', false);
        	var selectedOptions = 'mp3-'+el.options[el.selectedIndex].id+'-options';
        	displayClass(selectedOptions, true);
        }
        function onMp3UseQUalityChange(el){
            if (el.checked) {
                document.getElementById('mp3-quality-value').style.visibility = "";
            } else {
                document.getElementById('mp3-quality-value').style.visibility = "hidden";
            }
        }
        function onSubmitClick(){
            var fileName = getFileName('input-file')
            var ext = fileName.split('.')[1];
            var convert = document.getElementById('convert');
            convert.action = ext;
            convert.submit();
        }
    </script> 
</head>
<body>
    <h2>phono convert http</h1>
    <form id="convert" enctype="multipart/form-data" method="post">
    <div id="file">
        <input id="input-file" type="file" name="input-file" accept="{{.Accept}}" style="display:none" onchange="onInputFileChange()"/>
        <label id="input-file-label" for="input-file">select file</label>
    </div>
    <div id="output-format" style="display:none">
        output 
        {{range $key := .OutFormats}}
            <input type="radio" id="{{ $key }}" value="{{ $key }}" name="format" class="output-formats" onclick="onOutputFormatsClick(this)">
            <label for="{{ $key }}">{{ $key }}</label>
        {{end}}
    <br>
    </div>
    <div id="wav-options" class="output-options" style="display:none">
        bit depth
        <select name="wav-bit-depth">
            <option hidden disabled selected value>select</option>
            {{range $key, $value := .WavOptions.BitDepths}}
                <option value="{{ $key }}">{{ $value }}</option>
            {{end}}
        </select>
    <br>
    </div>
    <div id="mp3-options" class="output-options" style="display:none">
        channel mode
        <select name="mp3-channel-mode">
            <option hidden disabled selected value>select</option>
            {{range $key, $value := .Mp3Options.ChannelModes}}
                <option value="{{ $key }}">{{ $value }}</option>
            {{end}}
        </select>
        <br>
        bit rate mode
        <select id="mp3-bit-rate-mode" name="mp3-bit-rate-mode" onchange="onMp3BitRateModeChange(this)">
            <option hidden disabled selected value>select</option>
            {{range $key, $value := .Mp3Options.BitRateModes}}
                <option id="{{ $value }}" value="{{ printf "%d" $key }}">{{ $value }}</option>
            {{end}}
        </select>
        <br>
        <div class="mp3-bit-rate-mode-options mp3-{{ .Mp3Options.ABR }}-options mp3-{{ .Mp3Options.CBR }}-options" style="display:none">
            bit rate [8-320]
            <input type="text" name="mp3-bit-rate" maxlength="3" size="3">
        </div> 
        <div class="mp3-bit-rate-mode-options mp3-{{ .Mp3Options.VBR }}-options" style="display:none">
            vbr quality [0-10]
            <input type="text" name="mp3-vbr-quality" maxlength="2" size="3">
        </div>
        <div id="mp3-quality">
            <input type="checkbox" id="mp3-use-quality" name="mp3-use-quality" value="true" onchange="onMp3UseQUalityChange(this)">quality
            <div id="mp3-quality-value" style="display:inline;visibility:hidden">
                [0-10]
                <input type="text" name="mp3-quality" maxlength="2" size="3">
            </div>
            <br>  
        </div>
    </div>
    </form>
    <button id="submit" type="button" style="display:none" onclick="onSubmitClick()">convert</button> 
</body>
</html>
`
