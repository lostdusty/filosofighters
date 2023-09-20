package main

import (
	"archive/zip"
	"context"
	_ "embed"
	"io"
	"io/fs"
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

//go:embed assets/assets.zip
var assetsFile []byte

func init() {
	log.Println("\n@@@@@@@@@@@@@@@@@@@@@@@@@\n@ Filosofighters v1.0 @\n@@@@@@@@@@@@@@@@@@@@@@@@@")
	dlg, err := zenity.Progress(zenity.Title("Aguarde...."), zenity.Pulsate(), zenity.NoCancel())
	if err != nil {
		panic(err)
	} //Inicia o dialogo de progresso
	log.Println("Desempacotando arquivos base...")
	dlg.Text("Extraindo arquivos necessários")

	err = os.WriteFile("assets.zip", assetsFile, fs.FileMode(os.O_CREATE)) //Salva o arquivo em assets/assets.zip para o disco local
	if err != nil {
		zenity.Error("Não foi possível criar o arquivo\n"+err.Error(), zenity.Title("Falha"), zenity.ErrorIcon)
		panic(err)
	}
	err = Unzip("assets.zip", ".") //Descompacta o arquivo
	if err != nil {
		zenity.Error(err.Error(), zenity.Title("Impossivel continuar:"))
	}

	log.Println("Baixando Ruffle...")

	g := got.New()
	dlg.Text("Baixando o player de flash")
	err = g.Download("https://github.com/ruffle-rs/ruffle/releases/download/nightly-2023-09-20/ruffle-nightly-2023_09_20-windows-x86_64.zip", "ruffle.zip")
	if err != nil { //Baixa a última versão do Ruffle na data em que escrevo este comentário
		zenity.Error("Não foi possivel baixar o arquivo. Verifique a conexão com a rede", zenity.Title("Falha grave"), zenity.ErrorIcon)
		panic(err)
	}
	dlg.Text("Descompactando...")
	err = Unzip("ruffle.zip", ".") //Descompacta o arquivo baixado
	if err != nil {
		zenity.Error(err.Error(), zenity.Title("Impossivel continuar:"))
	}
	dlg.Close()
}

func main() {
	log.Println("Iniciando servidor HTTP...")
	httpServerExitDone := &sync.WaitGroup{} //Inicia o servidor HTTP

	httpServerExitDone.Add(1)                  //Adiciona o counter para o WaitGroup
	srv := startHttpServer(httpServerExitDone) //Inicia o servidor HTTP

	zenity.Info("Antes do jogo iniciar, nota importante:\n- Você pode jogar APENAS COM Karl Marx, Platão, Agostinho e Maquiavel.\n\n-Você pode jogar APENAS CONTRA Platão e Karl Max.\n\nIgnorar este aviso resultará em um game congelado.", zenity.Title("AVISO!"), zenity.WarningIcon)
	zenity.Notify("Antes do jogo iniciar, nota importante:\n- Você pode jogar APENAS COM Karl Marx, Platão, Agostinho e Maquiavel.\n\n-Você pode jogar APENAS CONTRA Platão e Karl Max.\n\nIgnorar este aviso resultará em um game congelado.", zenity.WarningIcon, zenity.Title("Aviso"))

	cmd := exec.Command("cmd", "/c", "ruffle.exe", "server/game.swf") //Executa o Ruffle com o jogo
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Stdout = io.Discard //Bug do jogo: Após a primeira batalha muitos erros são mostrados no console, degradando a performance, essa linha descarta o que for printado
	err := cmd.Run()
	if err != nil {
		zenity.Error("Não foi possível iniciar o Ruffle!\nImpossivel continuar", zenity.Title("Falha grave!"))
		panic(err)
	}

	if err := srv.Shutdown(context.TODO()); err != nil {
		zenity.Error("Não foi possível fechar o servidor HTTP!\n"+err.Error(), zenity.Title("Falha grave!"))
		panic(err)
	}

	httpServerExitDone.Wait() //Fecha o servidor HTTP
	log.Println("Servidor HTTP fechado.")

	os.Remove("ruffle.exe")
	os.Remove("LICENSE.md")
	os.Remove("assets.zip")
	os.Remove("ruffle.zip")
	os.RemoveAll("server") //Limpa todos os arquivos criados.
	log.Println("Limpeza concluída, saindo...")
}

func startHttpServer(wg *sync.WaitGroup) *http.Server {
	//Função que cria o servidor http com um WaitGroup. Isso permite que o servidor possa ser fechado em qualquer momento.
	http.Handle("/", http.FileServer(http.Dir("./server")))
	svr := &http.Server{
		Addr: ":4444",
	}

	go func() {
		defer wg.Done()

		if err := svr.ListenAndServe(); err != http.ErrServerClosed {
			zenity.Error("Não foi possível iniciar o servidor HTTP!\n"+err.Error(), zenity.Title("Falha grave!"))
			log.Fatalf("Erro do servidor HTTP: %v", err)
		}
	}()
	return svr
}

func Unzip(source, destination string) error {
	//Retirado de: https://gist.github.com/paulerickson/6d8650947ee4e3f3dbcc28fde10eaae7
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
