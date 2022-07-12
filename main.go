package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"runtime"
	"strings"
	"time"

	gl "github.com/go-gl/gl/v4.1-core/gl"
	"github.com/veandco/go-sdl2/sdl"
	//try use https://github.com/ungerik/go3d for maths
)

func compileShader(source string, shaderType uint32) (uint32, error) {
	shader := gl.CreateShader(shaderType)

	csources, free := gl.Strs(source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to compile %v: %v", source, log)
	}

	return shader, nil
}

func newProgram(vertexShaderSource, fragmentShaderSource string) (uint32, error) {
	vertexShader, err := compileShader(vertexShaderSource, gl.VERTEX_SHADER)
	if err != nil {
		return 0, err
	}

	fragmentShader, err := compileShader(fragmentShaderSource, gl.FRAGMENT_SHADER)
	if err != nil {
		return 0, err
	}

	program := gl.CreateProgram()

	gl.AttachShader(program, vertexShader)
	gl.AttachShader(program, fragmentShader)
	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		log := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(log))

		return 0, fmt.Errorf("failed to link program: %v", log)
	}

	gl.DeleteShader(vertexShader)
	gl.DeleteShader(fragmentShader)

	return program, nil
}

var uniRoll float32 = 0.0
var uniYaw float32 = 1.0
var uniPitch float32 = 0.0
var uniscale float32 = 0.3
var yrot float32 = 20.0
var zrot float32 = 0.0
var xrot float32 = 0.0
var UniScale int32

func main() {
	var window *sdl.Window
	var context sdl.GLContext
	var event sdl.Event
	var running bool
	var err error

	infoLogger := log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	//warnLogger := log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	criticalLogger := log.New(os.Stdout, "CRITICAL: ", log.Ldate|log.Ltime|log.Lshortfile)

	// event handling must run on the main OS thread
	runtime.LockOSThread()

	if err = sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		panic(err)
	}
	defer sdl.Quit()

	window, err = sdl.CreateWindow(winTitle, sdl.WINDOWPOS_UNDEFINED,
		sdl.WINDOWPOS_UNDEFINED,
		winWidth, winHeight, sdl.WINDOW_OPENGL)
	if err != nil {
		criticalLogger.Println(err)
		panic(err)
	}
	defer window.Destroy()

	sdl.GLSetAttribute(sdl.GL_RED_SIZE, 8)
	sdl.GLSetAttribute(sdl.GL_GREEN_SIZE, 8)
	sdl.GLSetAttribute(sdl.GL_BLUE_SIZE, 8)
	sdl.GLSetAttribute(sdl.GL_ALPHA_SIZE, 8)
	sdl.GLSetAttribute(sdl.GL_DOUBLEBUFFER, 1)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MAJOR_VERSION, 4)
	sdl.GLSetAttribute(sdl.GL_CONTEXT_MINOR_VERSION, 1)

	context, err = window.GLCreateContext()
	if err != nil {
		panic(err)
	}
	defer sdl.GLDeleteContext(context)

	// Initialize Glow
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	infoLogger.Println("OpenGL version", version)

	// Configure the vertex and fragment shaders
	program, err := newProgram(vertexShaderSource, fragmentShaderSource)
	if err != nil {
		panic(err)
	}

	gl.UseProgram(program)

	gl.Viewport(0, 0, int32(winWidth), int32(winHeight))

	// OPENGL FLAGS
	gl.ClearColor(0.0, 0.1, 0.0, 1.0)
	gl.Enable(gl.DEPTH_TEST)
	gl.DepthFunc(gl.LESS)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	//UNIFORM HOOK
	unistring := gl.Str("scaleMove\x00")
	UniScale = gl.GetUniformLocation(program, unistring)
	fmt.Printf("Uniform Link: %v\n", UniScale+1)

	// Configure the vertex data
	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)

	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, len(cubeVertices)*4, gl.Ptr(cubeVertices), gl.STATIC_DRAW)

	vertAttrib := uint32(gl.GetAttribLocation(program, gl.Str("Position\x00")))
	gl.EnableVertexAttribArray(vertAttrib)
	gl.VertexAttribPointerWithOffset(vertAttrib, 3, gl.FLOAT, false, 6*4, 0)

	colorAttrib := uint32(gl.GetAttribLocation(program, gl.Str("vertexColor\x00")))
	gl.EnableVertexAttribArray(colorAttrib)
	gl.VertexAttribPointerWithOffset(colorAttrib, 3, gl.FLOAT, false, 6*4, 3*4)

	running = true
	for running {
		for event = sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch t := event.(type) {
			case *sdl.QuitEvent:
				running = false
			case *sdl.MouseMotionEvent:
				xrot = float32(t.Y) / 2
				yrot = float32(t.X) / 2
				fmt.Printf("[%dms]MouseMotion\tid:%d\tx:%d\ty:%d\txrel:%d\tyrel:%d\n", t.Timestamp, t.Which, t.X, t.Y, t.XRel, t.YRel)
			case *sdl.JoyDeviceAddedEvent:
				enumerateGamepad()
			}
		}
		drawgl()
		window.GLSwap()
	}
}

func enumerateGamepad() {
	var numJoysticks int = sdl.NumJoysticks()

	for joystickId := 0; joystickId < numJoysticks; joystickId++ {
		if sdl.IsGameController(joystickId) {
			//controller := sdl.GameControllerOpen(joystickId)
			//controllerName := sdl.GameControllerNameForIndex(joystickId)
		}
	}
}

func drawgl() {
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)

	uniYaw = yrot * (math.Pi / 180.0)
	yrot = yrot - 1.0
	uniPitch = zrot * (math.Pi / 180.0)
	zrot = zrot - 0.5
	uniRoll = xrot * (math.Pi / 180.0)
	xrot = xrot - 0.2

	gl.Uniform4f(UniScale, float32(uniRoll), float32(uniYaw), float32(uniPitch), float32(uniscale))
	gl.Clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT)
	gl.DrawArrays(gl.TRIANGLES, int32(0), int32(6*2*3))

	time.Sleep(50 * time.Millisecond)

}

const (
	winTitle           = "OpenGL Shader"
	winWidth           = 640
	winHeight          = 480
	vertexShaderSource = `
	#version 330
	layout (location = 0) in vec3 Position;
	layout(location = 1) in vec3 vertexColor;
	uniform vec4 scaleMove;
	out vec3 fragmentColor;
	void main()
	{ 
	// YOU CAN OPTIMISE OUT cos(scaleMove.x) AND sin(scaleMove.y) AND UNIFORM THE VALUES IN
		vec3 scale = Position.xyz * scaleMove.w;
	// rotate on z pole
	vec3 rotatez = vec3((scale.x * cos(scaleMove.x) - scale.y * sin(scaleMove.x)), (scale.x * sin(scaleMove.x) + scale.y * cos(scaleMove.x)), scale.z);
	// rotate on y pole
		vec3 rotatey = vec3((rotatez.x * cos(scaleMove.y) - rotatez.z * sin(scaleMove.y)), rotatez.y, (rotatez.x * sin(scaleMove.y) + rotatez.z * cos(scaleMove.y)));
	// rotate on x pole
		vec3 rotatex = vec3(rotatey.x, (rotatey.y * cos(scaleMove.z) - rotatey.z * sin(scaleMove.z)), (rotatey.y * sin(scaleMove.z) + rotatey.z * cos(scaleMove.z)));
	// move
	vec3 move = vec3(rotatex.xy, rotatex.z - 0.2);
	// terrible perspective transform
	vec3 persp = vec3( move.x  / ( (move.z + 2) / 3 ) ,
			move.y  / ( (move.z + 2) / 3 ) ,
				move.z);

		gl_Position = vec4(persp, 1.0);
		fragmentColor = vertexColor;
	}
	`

	fragmentShaderSource = `
	#version 330
	out vec4 outColor;
	in vec3 fragmentColor;
	void main()
	{
		outColor = vec4(fragmentColor, 1.0);
	}
	`
)

var cubeVertices = []float32{
	//  X, Y, Z, R, G, B
	// Bottom
	-1.0, -1.0, -1.0, 0.0, 1.0, 1.0,
	1.0, -1.0, -1.0, 0.0, 1.0, 1.0,
	-1.0, -1.0, 1.0, 0.0, 1.0, 1.0,
	1.0, -1.0, -1.0, 0.0, 1.0, 1.0,
	1.0, -1.0, 1.0, 0.0, 1.0, 1.0,
	-1.0, -1.0, 1.0, 0.0, 1.0, 1.0,

	// Top
	-1.0, 1.0, -1.0, 1.0, 1.0, 0.0,
	-1.0, 1.0, 1.0, 1.0, 1.0, 0.0,
	1.0, 1.0, -1.0, 1.0, 1.0, 0.0,
	1.0, 1.0, -1.0, 1.0, 1.0, 0.0,
	-1.0, 1.0, 1.0, 1.0, 1.0, 0.0,
	1.0, 1.0, 1.0, 1.0, 1.0, 0.0,

	// Front
	-1.0, -1.0, 1.0, 0.5, 1.0, 0.5,
	1.0, -1.0, 1.0, 0.5, 1.0, 0.5,
	-1.0, 1.0, 1.0, 0.5, 1.0, 0.5,
	1.0, -1.0, 1.0, 0.5, 1.0, 0.5,
	1.0, 1.0, 1.0, 0.5, 1.0, 0.5,
	-1.0, 1.0, 1.0, 0.5, 1.0, 0.5,

	// Back
	-1.0, -1.0, -1.0, 1.0, 1.0, 0.0,
	-1.0, 1.0, -1.0, 1.0, 1.0, 0.0,
	1.0, -1.0, -1.0, 1.0, 1.0, 0.0,
	1.0, -1.0, -1.0, 1.0, 1.0, 0.0,
	-1.0, 1.0, -1.0, 1.0, 1.0, 0.0,
	1.0, 1.0, -1.0, 1.0, 1.0, 0.0,

	// Left
	-1.0, -1.0, 1.0, 1.0, 1.0, 0.0,
	-1.0, 1.0, -1.0, 1.0, 1.0, 0.0,
	-1.0, -1.0, -1.0, 1.0, 1.0, 0.0,
	-1.0, -1.0, 1.0, 1.0, 1.0, 0.0,
	-1.0, 1.0, 1.0, 1.0, 1.0, 0.0,
	-1.0, 1.0, -1.0, 1.0, 1.0, 0.0,

	// Right
	1.0, -1.0, 1.0, 0.0, 1.0, 1.0,
	1.0, -1.0, -1.0, 0.0, 1.0, 1.0,
	1.0, 1.0, -1.0, 0.0, 1.0, 1.0,
	1.0, -1.0, 1.0, 0.0, 1.0, 1.0,
	1.0, 1.0, -1.0, 0.0, 1.0, 1.0,
	1.0, 1.0, 1.0, 0.0, 1.0, 1.0,
}
