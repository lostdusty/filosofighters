package main

import (
	"archive/zip"
	"context"
	_ "embed"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/melbahja/got"
	"github.com/ncruces/zenity"
)

func init() {
	log.Println("Baixando arquivos base....")
	dlg, err := zenity.Progress(zenity.Title("Aguarde...."), zenity.Pulsate(), zenity.NoCancel())
	if err != nil {
		panic(err)
	}
	g := got.New()
	dlg.Text("Baixando arquivos necessários")
	err = g.Download("https://cdn.discordapp.com/attachments/653565137252515851/1153833686496718878/assets.zip", "assets.zip")
	if err != nil {
		zenity.Error("Não foi possivel baixar o arquivo. Verifique a conexão com a rede", zenity.Title("Falha grave"), zenity.ErrorIcon)
		panic(err)
	}
	dlg.Text("Descompactando...")
	err = Unzip("assets.zip", ".")
	if err != nil {
		zenity.Error(err.Error(), zenity.Title("Impossivel continuar:"))
	}
	dlg.Close()
}

func startHttpServer(wg *sync.WaitGroup) *http.Server {
	http.Handle("/", http.FileServer(http.Dir("./server")))
	svr := &http.Server{
		Addr: ":4444",
	}

	go func() {
		defer wg.Done()

		if err := svr.ListenAndServe(); err != http.ErrServerClosed {
			zenity.Error("Não foi possível iniciar o servidor HTTP!\n"+err.Error(), zenity.Title("Falha grave!"))
			log.Fatalf("ListenAndServe(): %v", err)
		}
	}()
	return svr
}

func main() {
	log.Println("Iniciando servidor HTTP...")
	httpServerExitDone := &sync.WaitGroup{}

	httpServerExitDone.Add(1)
	srv := startHttpServer(httpServerExitDone)

	zenity.Info("Antes do jogo iniciar, nota importante:\n- Você pode jogar APENAS COM Karl Marx, Platão, Agostinho e Maquiavel.\n\n-Você pode jogar APENAS CONTRA Platão e Karl Max.\n\nIgnorar este aviso resultará em um game congelado.", zenity.Title("AVISO!"), zenity.WarningIcon)
	zenity.Notify("Antes do jogo iniciar, nota importante:\n- Você pode jogar APENAS COM Karl Marx, Platão, Agostinho e Maquiavel.\n\n-Você pode jogar APENAS CONTRA Platão e Karl Max.\n\nIgnorar este aviso resultará em um game congelado.", zenity.WarningIcon, zenity.Title("Aviso"))

	cmd := exec.Command("cmd", "/c", "ruffle.exe", "server/game.swf")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Stdout = io.Discard
	err := cmd.Run()
	if err != nil {
		zenity.Error("Não foi possível iniciar o Ruffle!\nImpossivel continuar", zenity.Title("Falha grave!"))
		panic(err)
	}

	if err := srv.Shutdown(context.TODO()); err != nil {
		zenity.Error("Não foi possível fechar o servidor HTTP!\n"+err.Error(), zenity.Title("Falha grave!"))
		panic(err)
	}

	httpServerExitDone.Wait()
	log.Println("Servidor HTTP fechado.")

	os.Remove("ruffle.exe")
	os.RemoveAll("server")
	log.Println("Limpeza concluída, saindo...")
}

func Unzip(source, destination string) error {
	archive, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer archive.Close()
	for _, file := range archive.Reader.File {
		reader, err := file.Open()
		if err != nil {
			return err
		}
		defer reader.Close()
		path := filepath.Join(destination, file.Name)
		// Remove file if it already exists; no problem if it doesn't; other cases can error out below
		_ = os.Remove(path)
		// Create a directory at path, including parents
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
		// If file is _supposed_ to be a directory, we're done
		if file.FileInfo().IsDir() {
			continue
		}
		// otherwise, remove that directory (_not_ including parents)
		err = os.Remove(path)
		if err != nil {
			return err
		}
		// and create the actual file.  This ensures that the parent directories exist!
		// An archive may have a single file with a nested path, rather than a file for each parent dir
		writer, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}
		defer writer.Close()
		_, err = io.Copy(writer, reader)
		if err != nil {
			return err
		}
	}
	return nil
}
